package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

const (
	// rcServiceName is the sprite-env service that runs `claude remote-control`.
	rcServiceName = "claude-rc"
	// rcScriptPath is the launcher script the service executes, written on start.
	rcScriptPath = "/root/.sproot/claude-rc.sh"
	// rcCredsPath is where Claude Code caches the claude.ai OAuth credential after
	// an interactive `claude auth login`. Remote Control requires it; the inference
	// tokens sproot forwards (CLAUDE_CODE_OAUTH_TOKEN, API keys) are not accepted.
	rcCredsPath = "/root/.claude/.credentials.json"
)

// RCOptions controls the behavior of RunRC.
type RCOptions struct {
	Name        string
	Dir         string        // working directory for the session; empty means the home directory
	Spawn       string        // same-dir | worktree | session
	SessionName string        // session title in the web/mobile UI; empty means the sprite name
	Close       bool          // stop the session instead of starting it
	client      SpritesClient // nil: constructed from token at runtime
}

// RunRC starts or stops a Claude Code Remote Control session on the named sprite.
//
// Remote Control runs as a sprite-env service: it dials out to Anthropic (no
// inbound port) and appears in the operator's claude.ai/code and mobile session
// list by name. As a service it keeps the sprite awake and persists across reboots
// until closed. The credential is provided by a one-time interactive
// `claude auth login` on the sprite (run via `sproot console`), cached at
// rcCredsPath, and reused on later starts.
func RunRC(ctx context.Context, opts RCOptions) error {
	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return err
	}

	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	handle := client.GetHandle(opts.Name)
	l := log.Stderr()

	if opts.Close {
		return rcClose(handle, opts.Name, l)
	}
	return rcStart(handle, opts, l)
}

func rcStart(handle SpriteHandle, opts RCOptions, l *log.Logger) error {
	mode := opts.Spawn
	if mode == "" {
		mode = "same-dir"
	}
	switch mode {
	case "same-dir", "worktree", "session":
	default:
		return fmt.Errorf("invalid --spawn %q (want same-dir, worktree, or session)", mode)
	}

	if _, err := handle.ReadFile(rcCredsPath); err != nil {
		return fmt.Errorf("sprite %q is not authenticated for Claude Remote Control (no %s); "+
			"run `sproot console %s` and complete `claude auth login` once, then re-run `sproot rc %s`",
			opts.Name, rcCredsPath, opts.Name, opts.Name)
	}

	title := opts.SessionName
	if title == "" {
		title = opts.Name
	}

	script := buildRCScript(opts.Dir, title, mode)
	if err := handle.WriteFile(rcScriptPath, []byte(script), 0o755); err != nil {
		return fmt.Errorf("write launcher script: %w", err)
	}

	body, err := json.Marshal(map[string]any{"cmd": rcScriptPath})
	if err != nil {
		return fmt.Errorf("marshal service body: %w", err)
	}
	if err := handle.RunCommand("sprite-env",
		[]string{"curl", "-X", "PUT", "/v1/services/" + rcServiceName, "-d", string(body)},
		nil, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("register remote-control service: %w", err)
	}

	l.Successf("Claude Remote Control started on %s (service %q)", opts.Name, rcServiceName)
	l.Infof("open https://claude.ai/code or the Claude mobile app and select the session named %q", title)
	l.Infof("the sprite stays awake until you run: sproot rc %s --close", opts.Name)
	return nil
}

func rcClose(handle SpriteHandle, name string, l *log.Logger) error {
	if err := handle.RunCommand("sprite-env",
		[]string{"curl", "-X", "DELETE", "/v1/services/" + rcServiceName},
		nil, os.Stdout, os.Stderr); err != nil {
		return fmt.Errorf("delete remote-control service: %w", err)
	}
	l.Successf("Claude Remote Control stopped on %s; the sprite will pause once idle", name)
	return nil
}

// buildRCScript renders the launcher the sprite-env service runs. It sources the
// login profiles so `claude` (installed under ~/.local/bin by the nix/npm modules)
// is on PATH, changes to the working directory, then execs remote-control.
func buildRCScript(dir, title, mode string) string {
	workdir := rcWorkdirExpr(dir)
	safeTitle := strings.ReplaceAll(title, `"`, "'")
	return fmt.Sprintf(`#!/usr/bin/env bash
set -e
. ~/.profile 2>/dev/null || true
. ~/.bashrc 2>/dev/null || true
cd "%s"
exec claude remote-control --name "%s" --spawn %s
`, workdir, safeTitle, mode)
}

// rcWorkdirExpr returns a shell expression for the working directory, mapping a
// leading ~ to $HOME (expanded inside the double-quoted cd in buildRCScript).
func rcWorkdirExpr(dir string) string {
	switch {
	case dir == "" || dir == "~":
		return "$HOME"
	case strings.HasPrefix(dir, "~/"):
		return "$HOME/" + dir[2:]
	default:
		return dir
	}
}
