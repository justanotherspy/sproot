package modules

// corepack enables Node.js corepack and pre-activates the listed package managers.
//
//	- type: corepack
//	  corepack:
//	    managers: [pnpm, yarn]

import (
	"fmt"
	"os/exec"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("corepack", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &corepackPhase{cfg: cfg.Corepack}, nil
	})
}

type corepackPhase struct {
	cfg *config.CorepackConfig
}

func (p *corepackPhase) Type() string { return "corepack" }
func (p *corepackPhase) Name() string { return "corepack" }

func (p *corepackPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, mgr := range p.cfg.Managers {
		if _, err := exec.LookPath(mgr); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *corepackPhase) Run(ctx *phase.Context) error {
	if err := runCmd(ctx.Log, "corepack", "enable"); err != nil {
		return err
	}
	for _, mgr := range p.cfg.Managers {
		if err := runCmd(ctx.Log, "corepack", "prepare", mgr+"@latest", "--activate"); err != nil {
			return fmt.Errorf("corepack prepare %s: %w", mgr, err)
		}
	}
	return nil
}

func (p *corepackPhase) Verify(_ *phase.Context) error {
	for _, mgr := range p.cfg.Managers {
		if _, err := exec.LookPath(mgr); err != nil {
			return fmt.Errorf("corepack: %q not on PATH after enable", mgr)
		}
	}
	return nil
}
