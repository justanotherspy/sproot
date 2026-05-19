package modules

// git_identity configures git user identity, global settings, aliases, and SSH commit signing.
// It reads identity fields from ctx.Identity; no per-phase YAML fields are needed.
//
//	- type: git_identity

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("git_identity", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &gitIdentityPhase{}, nil
	})
}

type gitIdentityPhase struct{}

func (p *gitIdentityPhase) Type() string { return "git_identity" }
func (p *gitIdentityPhase) Name() string { return "git_identity" }

func (p *gitIdentityPhase) ShouldRun(ctx *phase.Context) (bool, error) {
	id := ctx.Identity
	name, _ := outputOf("git", "config", "--global", "user.name")
	email, _ := outputOf("git", "config", "--global", "user.email")
	branch, _ := outputOf("git", "config", "--global", "init.defaultBranch")
	gpgFmt, _ := outputOf("git", "config", "--global", "gpg.format")

	if email == "noreply@sprites.dev" {
		return true, nil
	}
	if name != id.GitUserName || email != id.GitUserEmail || branch != id.GitDefaultBranch {
		return true, nil
	}
	if gpgFmt != "ssh" {
		return true, nil
	}
	return false, nil
}

func (p *gitIdentityPhase) Run(ctx *phase.Context) error {
	id := ctx.Identity
	settings := [][]string{
		{"user.name", id.GitUserName},
		{"user.email", id.GitUserEmail},
		{"init.defaultBranch", id.GitDefaultBranch},
		{"pull.rebase", "true"},
		{"push.autoSetupRemote", "true"},
		{"rerere.enabled", "true"},
		{"color.ui", "auto"},
		{"core.editor", "vim"},
		{"fetch.prune", "true"},
		{"alias.lg", "log --oneline --graph --decorate --all"},
		{"alias.last", "log -1 HEAD"},
		{"alias.amend", "commit --amend --no-edit"},
		{"alias.unstage", "reset HEAD --"},
		{"alias.cleanb", "!git branch --merged | grep -vE '^\\*|^.\\s*(main|master|develop)$' | xargs -r git branch -d"},
	}
	for _, kv := range settings {
		if err := runCmd(ctx.Log, "git", "config", "--global", kv[0], kv[1]); err != nil {
			return err
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	pub := filepath.Join(home, ".ssh", "id_ed25519.pub")
	if _, err := os.Stat(pub); err == nil {
		signers := filepath.Join(home, ".ssh", "allowed_signers")
		signingSettings := [][]string{
			{"gpg.format", "ssh"},
			{"user.signingkey", pub},
			{"commit.gpgsign", "true"},
			{"tag.gpgsign", "true"},
			{"gpg.ssh.allowedSignersFile", signers},
		}
		for _, kv := range signingSettings {
			if err := runCmd(ctx.Log, "git", "config", "--global", kv[0], kv[1]); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p *gitIdentityPhase) Verify(ctx *phase.Context) error {
	id := ctx.Identity
	name, err := outputOf("git", "config", "--global", "user.name")
	if err != nil || name != id.GitUserName {
		return fmt.Errorf("git_identity: user.name is %q, want %q", name, id.GitUserName)
	}
	email, err := outputOf("git", "config", "--global", "user.email")
	if err != nil || email != id.GitUserEmail {
		return fmt.Errorf("git_identity: user.email is %q, want %q", email, id.GitUserEmail)
	}
	return nil
}
