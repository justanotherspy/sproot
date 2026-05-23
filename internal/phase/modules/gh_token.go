package modules

// gh_token authenticates the gh CLI using GH_TOKEN from the sprite environment.
// The token is expected to be present (injected via the env block in sproot.yaml)
// and is piped to gh auth login so that credentials persist in
// ~/.config/gh/hosts.yml for all future sessions without needing GH_TOKEN set.
//
//	- type: gh_token

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("gh_token", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &ghTokenPhase{}, nil
	})
}

type ghTokenPhase struct{}

func (p *ghTokenPhase) Type() string { return "gh_token" }
func (p *ghTokenPhase) Name() string { return "gh_token" }

func (p *ghTokenPhase) ShouldRun(ctx *phase.Context) (bool, error) {
	if os.Getenv("GH_TOKEN") == "" {
		ctx.Log.Warn("gh_token: GH_TOKEN not set, skipping")
		return false, nil
	}
	if !checkCmd("gh", "auth", "status", "-h", "github.com") {
		return true, nil
	}
	who, _ := outputOf("gh", "api", "user", "--jq", ".login")
	if who != ctx.Identity.GHUsername {
		return true, nil
	}
	statusOut, _ := outputOf("gh", "auth", "status", "-h", "github.com")
	if !strings.Contains(statusOut, "admin:public_key") ||
		!strings.Contains(statusOut, "admin:ssh_signing_key") {
		return true, nil
	}
	return false, nil
}

func (p *ghTokenPhase) Run(ctx *phase.Context) error {
	token := os.Getenv("GH_TOKEN")
	if token == "" {
		ctx.Log.Warn("gh_token: GH_TOKEN not set, skipping GitHub authentication")
		return nil
	}

	cmd := ghLoginCmd(token)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh_token: gh auth login: %w", err)
	}
	ctx.Log.Info("gh authenticated")
	return nil
}

// ghLoginCmd constructs the gh auth login command with token piped via stdin.
// GH_TOKEN/GITHUB_TOKEN are stripped from the child environment: gh refuses to
// run `auth login` while either is set (it would use the env value instead of
// persisting credentials) and exits non-zero. With them cleared, login writes
// the token to ~/.config/gh/hosts.yml so future sessions authenticate without
// the env var present.
func ghLoginCmd(token string) *exec.Cmd {
	cmd := exec.Command("gh", "auth", "login",
		"--hostname", "github.com",
		"--git-protocol", "ssh",
		"--with-token",
	)
	cmd.Env = envWithout(os.Environ(), "GH_TOKEN", "GITHUB_TOKEN")
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd
}

// envWithout returns env with any "KEY=value" entry whose key matches one of
// keys removed. Used to clear auth env vars that would otherwise short-circuit
// commands that need to operate on persisted credentials.
func envWithout(env []string, keys ...string) []string {
	drop := make(map[string]bool, len(keys))
	for _, k := range keys {
		drop[k] = true
	}
	out := env[:0:0]
	for _, e := range env {
		name, _, _ := strings.Cut(e, "=")
		if drop[name] {
			continue
		}
		out = append(out, e)
	}
	return out
}

func (p *ghTokenPhase) Verify(ctx *phase.Context) error {
	if os.Getenv("GH_TOKEN") == "" {
		return nil
	}
	if !checkCmd("gh", "auth", "status", "-h", "github.com") {
		return fmt.Errorf("gh_token: gh auth status failed after login")
	}
	who, err := outputOf("gh", "api", "user", "--jq", ".login")
	if err != nil {
		return fmt.Errorf("gh_token: could not get logged-in user: %w", err)
	}
	if who != ctx.Identity.GHUsername {
		return fmt.Errorf("gh_token: logged in as %q, want %q", who, ctx.Identity.GHUsername)
	}
	return nil
}
