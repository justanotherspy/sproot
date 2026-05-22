package modules

// uv_tool installs Python tools via uv tool install.
// uv is installed automatically to /usr/local/bin if not already on PATH.
//
//	- type: uv_tool
//	  tools:
//	    - name: ruff
//	    - name: pre-commit
//	    - name: g            # binary name
//	      pkg: garlic        # PyPI package name (when it differs from name)

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
	if _, err := exec.LookPath("uv"); err != nil {
		return true, nil
	}
	for _, t := range p.cfg.Tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			return true, nil
		}
	}
	return false, nil
}

func (p *uvToolPhase) Run(ctx *phase.Context) error {
	if _, err := exec.LookPath("uv"); err != nil {
		if err := runCmd(ctx.Log, "sh", "-c",
			"curl -LsSf https://astral.sh/uv/install.sh | UV_INSTALL_DIR=/usr/local/bin sh"); err != nil {
			return fmt.Errorf("uv_tool: install uv: %w", err)
		}
	}
	for _, t := range p.cfg.Tools {
		pkg := t.Pkg
		if pkg == "" {
			pkg = t.Name
		}
		if err := runCmd(ctx.Log, "uv", "tool", "install", pkg); err != nil {
			return err
		}
	}
	return nil
}

func (p *uvToolPhase) Verify(_ *phase.Context) error {
	if _, err := exec.LookPath("uv"); err != nil {
		return fmt.Errorf("uv_tool: uv not on PATH after install")
	}
	for _, t := range p.cfg.Tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			return fmt.Errorf("uv_tool: %q not on PATH after install", t.Name)
		}
	}
	return nil
}
