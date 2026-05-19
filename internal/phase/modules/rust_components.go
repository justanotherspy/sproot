package modules

// rust_components pins the stable Rust toolchain and installs the listed components.
//
//	- type: rust_components
//	  rust_components:
//	    components: [clippy, rustfmt, rust-analyzer]

import (
	"fmt"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("rust_components", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &rustComponentsPhase{cfg: cfg.RustComponents}, nil
	})
}

type rustComponentsPhase struct {
	cfg *config.RustComponentsConfig
}

func (p *rustComponentsPhase) Type() string { return "rust_components" }
func (p *rustComponentsPhase) Name() string { return "rust_components" }

func (p *rustComponentsPhase) ShouldRun(_ *phase.Context) (bool, error) {
	installed, err := outputOf("rustup", "component", "list", "--installed")
	if err != nil {
		return true, nil
	}
	for _, c := range p.cfg.Components {
		if !strings.Contains(installed, c) {
			return true, nil
		}
	}
	return false, nil
}

func (p *rustComponentsPhase) Run(ctx *phase.Context) error {
	if err := runCmd(ctx.Log, "rustup", "default", "stable"); err != nil {
		return err
	}
	args := append([]string{"component", "add"}, p.cfg.Components...)
	return runCmd(ctx.Log, "rustup", args...)
}

func (p *rustComponentsPhase) Verify(_ *phase.Context) error {
	installed, err := outputOf("rustup", "component", "list", "--installed")
	if err != nil {
		return fmt.Errorf("rust_components: rustup not available: %w", err)
	}
	for _, c := range p.cfg.Components {
		if !strings.Contains(installed, c) {
			return fmt.Errorf("rust_components: component %q not installed", c)
		}
	}
	return nil
}
