package modules

// shell_completion generates and installs shell completion scripts for one or
// more commands into per-user completion directories, then wires zsh's fpath so
// it loads them. bash and fish auto-load from their standard user directories;
// only zsh needs the rc block.
//
//	- type: shell_completion
//	  completions:
//	    - command: sproot
//	      shells: [bash, zsh, fish]
//	    - command: gh
//	      shells: [bash, zsh]
//	      gen: "{command} completion {shell}"   # optional; this is the default

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const (
	zshCompBegin = "# BEGIN SPROOT COMPLETIONS"
	zshCompEnd   = "# END SPROOT COMPLETIONS"

	// defaultCompletionGen is the cobra convention used by sproot, gh, kubectl,
	// and most modern CLIs. Override per entry via the gen field for tools that
	// emit completions differently.
	defaultCompletionGen = "{command} completion {shell}"
)

// zshCompletionBlock adds the sproot completion dir to fpath and initializes the
// completion system. Written to ~/.zshrc when any entry targets zsh. It ends in a
// newline so its stored block hash matches applyManagedBlock's idempotency check.
const zshCompletionBlock = "fpath=(~/.zfunc $fpath)\nautoload -Uz compinit && compinit\n"

func init() {
	phase.Register("shell_completion", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.ShellCompletion == nil {
			return nil, fmt.Errorf("shell_completion: config is nil")
		}
		return &shellCompletionPhase{cfg: cfg.ShellCompletion}, nil
	})
}

type shellCompletionPhase struct {
	cfg *config.ShellCompletionConfig
}

func (p *shellCompletionPhase) Type() string { return "shell_completion" }
func (p *shellCompletionPhase) Name() string { return "shell_completion" }

// completionDest returns the per-user install path for command's completion in
// shell. All paths are user-writable, so no privilege escalation is needed.
func completionDest(command, shell string) (string, error) {
	switch shell {
	case "bash":
		return expandTilde(filepath.Join("~/.local/share/bash-completion/completions", command))
	case "zsh":
		return expandTilde(filepath.Join("~/.zfunc", "_"+command))
	case "fish":
		return expandTilde(filepath.Join("~/.config/fish/completions", command+".fish"))
	default:
		return "", fmt.Errorf("shell_completion: unsupported shell %q", shell)
	}
}

func (p *shellCompletionPhase) usesZsh() bool {
	for _, e := range p.cfg.Completions {
		for _, sh := range e.Shells {
			if sh == "zsh" {
				return true
			}
		}
	}
	return false
}

func (p *shellCompletionPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, e := range p.cfg.Completions {
		for _, sh := range e.Shells {
			dest, err := completionDest(e.Command, sh)
			if err != nil {
				return false, err
			}
			if _, err := os.Stat(dest); err != nil {
				return true, nil
			}
		}
	}
	if p.usesZsh() && !zshBlockCurrent() {
		return true, nil
	}
	return false, nil
}

func (p *shellCompletionPhase) Run(ctx *phase.Context) error {
	for _, e := range p.cfg.Completions {
		gen := e.Gen
		if gen == "" {
			gen = defaultCompletionGen
		}
		for _, sh := range e.Shells {
			script, err := generateCompletion(gen, e.Command, sh)
			if err != nil {
				return fmt.Errorf("shell_completion: generate %s/%s: %w", e.Command, sh, err)
			}
			dest, err := completionDest(e.Command, sh)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return fmt.Errorf("shell_completion: create %s: %w", filepath.Dir(dest), err)
			}
			if err := os.WriteFile(dest, script, 0o644); err != nil {
				return fmt.Errorf("shell_completion: write %s: %w", dest, err)
			}
			ctx.Log.Infof("installed %s completion for %s -> %s", sh, e.Command, dest)
		}
	}
	if p.usesZsh() {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("shell_completion: %w", err)
		}
		zshrc := filepath.Join(home, ".zshrc")
		if err := applyManagedBlock(zshrc, zshCompletionBlock, zshCompBegin, zshCompEnd); err != nil {
			return fmt.Errorf("shell_completion: wire zsh fpath: %w", err)
		}
		ctx.Log.Infof("wired zsh completion fpath in %s", zshrc)
	}
	return nil
}

func (p *shellCompletionPhase) Verify(_ *phase.Context) error {
	for _, e := range p.cfg.Completions {
		for _, sh := range e.Shells {
			dest, err := completionDest(e.Command, sh)
			if err != nil {
				return err
			}
			info, err := os.Stat(dest)
			if err != nil {
				return fmt.Errorf("shell_completion: %s/%s not installed at %s: %w", e.Command, sh, dest, err)
			}
			if info.Size() == 0 {
				return fmt.Errorf("shell_completion: %s/%s is empty at %s", e.Command, sh, dest)
			}
		}
	}
	if p.usesZsh() && !zshBlockCurrent() {
		return fmt.Errorf("shell_completion: zsh fpath block missing or stale in ~/.zshrc")
	}
	return nil
}

// zshBlockCurrent reports whether ~/.zshrc carries the current sproot completion
// fpath block.
func zshBlockCurrent() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	content, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		return false
	}
	return extractManagedBlockHash(string(content), zshCompBegin, zshCompEnd) == blockHash(zshCompletionBlock)
}

// generateCompletion renders gen (tokens {command} and {shell}), splits it into
// argv, executes it, and returns the script written to stdout.
func generateCompletion(gen, command, shell string) ([]byte, error) {
	rendered := strings.NewReplacer("{command}", command, "{shell}", shell).Replace(gen)
	fields := strings.Fields(rendered)
	if len(fields) == 0 {
		return nil, fmt.Errorf("gen command is empty")
	}
	out, err := exec.Command(fields[0], fields[1:]...).Output() // nosemgrep
	if err != nil {
		return nil, fmt.Errorf("%q: %w", rendered, err)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("%q produced no output", rendered)
	}
	return out, nil
}
