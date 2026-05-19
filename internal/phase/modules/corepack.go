package modules

// corepack enables Node.js corepack and pre-activates pnpm and yarn.
//
//	- type: corepack

import (
	"fmt"
	"os/exec"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("corepack", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &corepackPhase{}, nil
	})
}

type corepackPhase struct{}

func (p *corepackPhase) Type() string { return "corepack" }
func (p *corepackPhase) Name() string { return "corepack" }

func (p *corepackPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, bin := range []string{"pnpm", "yarn"} {
		if _, err := exec.LookPath(bin); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *corepackPhase) Run(ctx *phase.Context) error {
	if err := runCmd(ctx.Log, "corepack", "enable"); err != nil {
		return err
	}
	for _, pkg := range []string{"pnpm@latest", "yarn@latest"} {
		if err := runCmd(ctx.Log, "corepack", "prepare", pkg, "--activate"); err != nil {
			return fmt.Errorf("corepack prepare %s: %w", pkg, err)
		}
	}
	return nil
}

func (p *corepackPhase) Verify(_ *phase.Context) error {
	for _, bin := range []string{"pnpm", "yarn"} {
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("corepack: %q not on PATH after enable", bin)
		}
	}
	return nil
}
