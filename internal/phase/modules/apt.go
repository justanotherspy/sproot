package modules

// apt installs system packages via apt-get and optionally creates symlinks.
//
//	- type: apt
//	  packages: [git, vim, jq]
//	  symlinks:
//	    - from: /usr/bin/batcat
//	      to: /usr/local/bin/bat

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("apt", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Apt == nil {
			return nil, fmt.Errorf("apt: config is nil")
		}
		return &aptPhase{cfg: cfg.Apt}, nil
	})
}

type aptPhase struct {
	cfg *config.AptConfig
}

func (p *aptPhase) Type() string { return "apt" }
func (p *aptPhase) Name() string { return "apt" }

func (p *aptPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, pkg := range p.cfg.Packages {
		if !checkCmd("dpkg", "-s", pkg) {
			return true, nil
		}
	}
	for _, s := range p.cfg.Symlinks {
		to, err := expandTilde(s.To)
		if err != nil {
			return false, err
		}
		if _, err := os.Lstat(to); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *aptPhase) Run(ctx *phase.Context) error {
	if len(p.cfg.Packages) > 0 {
		args := append([]string{"install", "-y"}, p.cfg.Packages...)
		if err := runPrivileged(ctx.Log, "apt-get", args...); err != nil {
			return err
		}
	}
	for _, s := range p.cfg.Symlinks {
		to, err := expandTilde(s.To)
		if err != nil {
			return err
		}
		from, err := expandTilde(s.From)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
			return fmt.Errorf("apt: mkdir for symlink %s: %w", to, err)
		}
		if err := runCmd(ctx.Log, "ln", "-sf", from, to); err != nil {
			return fmt.Errorf("apt: symlink %s -> %s: %w", to, from, err)
		}
	}
	return nil
}

func (p *aptPhase) Verify(_ *phase.Context) error {
	for _, pkg := range p.cfg.Packages {
		if !checkCmd("dpkg", "-s", pkg) {
			return fmt.Errorf("apt: package %q is not installed", pkg)
		}
	}
	for _, s := range p.cfg.Symlinks {
		to, err := expandTilde(s.To)
		if err != nil {
			return err
		}
		if _, err := os.Lstat(to); err != nil {
			return fmt.Errorf("apt: symlink %q missing after install", s.To)
		}
	}
	return nil
}
