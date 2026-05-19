package modules

// gh_token authenticates the gh CLI using a GitHub PAT injected by the host CLI.
// The token is read from the SPRITE_GH_TOKEN environment variable, passed to
// gh auth login, and immediately unset -- it is never written to disk.
//
// The host CLI (sproot new, Phase 5) must supply SPRITE_GH_TOKEN before running
// sproot setup. Required scopes: admin:public_key, admin:ssh_signing_key.
//
// TODO(phase-5): SPRITE_GH_TOKEN must be injected by the host CLI via
// sprite.Command(...).Env("SPRITE_GH_TOKEN", token) before this module runs.
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
	token := os.Getenv("SPRITE_GH_TOKEN")
	if token == "" {
		return fmt.Errorf("gh_token: SPRITE_GH_TOKEN is not set; " +
			"pass it via sproot new or export it manually before running setup")
	}

	// Feed token via stdin to avoid it appearing in process args.
	cmd := ghLoginCmd(token)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh_token: gh auth login: %w", err)
	}
	_ = os.Unsetenv("SPRITE_GH_TOKEN")
	ctx.Log.Info("gh authenticated")
	return nil
}

// ghLoginCmd constructs the gh auth login command with token piped via stdin.
func ghLoginCmd(token string) *exec.Cmd {
	cmd := exec.Command("gh", "auth", "login",
		"--hostname", "github.com",
		"--git-protocol", "ssh",
		"--with-token",
	)
	cmd.Stdin = strings.NewReader(token)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd
}

func (p *ghTokenPhase) Verify(ctx *phase.Context) error {
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
