package modules

// git_identity configures git user.name, user.email, init.defaultBranch, and SSH
// commit signing from the top-level identity block. An optional config map allows
// the config repo to set arbitrary git config keys without hardcoding them here.
//
//	- type: git_identity
//	  git_identity:
//	    config:
//	      pull.rebase: "true"
//	      core.editor: vim

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("git_identity", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &gitIdentityPhase{cfg: cfg.GitIdentity}, nil
	})
}

type gitIdentityPhase struct {
	cfg *config.GitIdentityConfig
}

func (p *gitIdentityPhase) Type() string { return "git_identity" }
func (p *gitIdentityPhase) Name() string { return "git_identity" }

func (p *gitIdentityPhase) ShouldRun(ctx *phase.Context) (bool, error) {
	id := ctx.Identity
	name, _ := outputOf("git", "config", "--global", "user.name")
	email, _ := outputOf("git", "config", "--global", "user.email")
	branch, _ := outputOf("git", "config", "--global", "init.defaultBranch")

	if email == "noreply@sprites.dev" {
		return true, nil
	}
	if name != id.GitUserName || email != id.GitUserEmail || branch != id.GitDefaultBranch {
		return true, nil
	}

	for k, want := range p.cfg.Config {
		got, _ := outputOf("git", "config", "--global", k)
		if got != want {
			return true, nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return true, nil
	}
	pub := filepath.Join(home, ".ssh", "id_ed25519.pub")
	if _, err := os.Stat(pub); err == nil {
		gpgFmt, _ := outputOf("git", "config", "--global", "gpg.format")
		if gpgFmt != "ssh" {
			return true, nil
		}
	}
	return false, nil
}

func (p *gitIdentityPhase) Run(ctx *phase.Context) error {
	id := ctx.Identity
	settings := [][]string{
		{"user.name", id.GitUserName},
		{"user.email", id.GitUserEmail},
		{"init.defaultBranch", id.GitDefaultBranch},
	}
	for k, v := range p.cfg.Config {
		settings = append(settings, []string{k, v})
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
