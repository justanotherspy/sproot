package modules

// npm runs npm install in a target directory.
//
//	- type: npm
//	  dir: ~/my-project

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("npm", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Npm == nil {
			return nil, fmt.Errorf("npm: config is nil")
		}
		return &npmPhase{cfg: cfg.Npm}, nil
	})
}

type npmPhase struct {
	cfg *config.NpmConfig
}

func (p *npmPhase) Type() string { return "npm" }
func (p *npmPhase) Name() string { return "npm" }

func (p *npmPhase) ShouldRun(_ *phase.Context) (bool, error) {
	dir, err := expandTilde(p.cfg.Dir)
	if err != nil {
		return true, nil
	}
	_, err = os.Stat(filepath.Join(dir, "node_modules"))
	return os.IsNotExist(err), nil
}

func (p *npmPhase) Run(ctx *phase.Context) error {
	dir, err := expandTilde(p.cfg.Dir)
	if err != nil {
		return fmt.Errorf("npm: expand dir: %w", err)
	}
	return runCmdIn(dir, ctx.Log, "npm", "install")
}

func (p *npmPhase) Verify(_ *phase.Context) error {
	dir, err := expandTilde(p.cfg.Dir)
	if err != nil {
		return fmt.Errorf("npm: expand dir: %w", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules")); err != nil {
		return fmt.Errorf("npm: node_modules missing in %s after install", dir)
	}
	return nil
}
