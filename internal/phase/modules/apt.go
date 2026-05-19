package modules

// apt installs system packages via apt-get.
//
//	- type: apt
//	  packages: [git, vim, jq]

import (
	"fmt"

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
	return false, nil
}

func (p *aptPhase) Run(ctx *phase.Context) error {
	args := append([]string{"install", "-y"}, p.cfg.Packages...)
	return runCmd(ctx.Log, "apt-get", args...)
}

func (p *aptPhase) Verify(_ *phase.Context) error {
	for _, pkg := range p.cfg.Packages {
		if !checkCmd("dpkg", "-s", pkg) {
			return fmt.Errorf("apt: package %q is not installed", pkg)
		}
	}
	return nil
}
