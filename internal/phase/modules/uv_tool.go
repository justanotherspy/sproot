package modules

// uv_tool installs Python tools via uv tool install.
//
//	- type: uv_tool
//	  tools:
//	    - name: ruff
//	    - name: pre-commit

import (
	"fmt"
	"os/exec"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("uv_tool", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.UVTool == nil {
			return nil, fmt.Errorf("uv_tool: config is nil")
		}
		return &uvToolPhase{cfg: cfg.UVTool}, nil
	})
}

type uvToolPhase struct {
	cfg *config.UVToolConfig
}

func (p *uvToolPhase) Type() string { return "uv_tool" }
func (p *uvToolPhase) Name() string { return "uv_tool" }

func (p *uvToolPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, t := range p.cfg.Tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *uvToolPhase) Run(ctx *phase.Context) error {
	for _, t := range p.cfg.Tools {
		if err := runCmd(ctx.Log, "uv", "tool", "install", t.Name); err != nil {
			return err
		}
	}
	return nil
}

func (p *uvToolPhase) Verify(_ *phase.Context) error {
	for _, t := range p.cfg.Tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			return fmt.Errorf("uv_tool: %q not on PATH after install", t.Name)
		}
	}
	return nil
}
