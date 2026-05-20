// Package modules contains concrete Phase implementations.
// Each module file registers itself via Register in its init function.
//
// Import this package blank to activate all registered module factories:
//
//	import _ "github.com/justanotherspy/sproot/internal/phase/modules"
package modules

// cmd runs an arbitrary shell command with an optional idempotency check.
//
//	- type: cmd
//	  run: "echo hello"
//	  check: "true"   # optional; omit to always run

import (
	"fmt"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("cmd", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Cmd == nil {
			return nil, fmt.Errorf("cmd: config is nil")
		}
		return &cmdPhase{cfg: cfg.Cmd}, nil
	})
}

type cmdPhase struct {
	cfg *config.CmdConfig
}

func (p *cmdPhase) Type() string { return "cmd" }
func (p *cmdPhase) Name() string {
	if p.cfg.Name != "" {
		return fmt.Sprintf("cmd(%s)", p.cfg.Name)
	}
	return "cmd"
}

func (p *cmdPhase) ShouldRun(_ *phase.Context) (bool, error) {
	if p.cfg.Check == "" {
		return true, nil
	}
	return !checkCmd("sh", "-c", p.cfg.Check), nil
}

func (p *cmdPhase) Run(ctx *phase.Context) error {
	return runCmd(ctx.Log, "sh", "-c", p.cfg.Run)
}

func (p *cmdPhase) Verify(_ *phase.Context) error {
	if p.cfg.Check == "" {
		return nil
	}
	if !checkCmd("sh", "-c", p.cfg.Check) {
		return fmt.Errorf("cmd: check command failed after run: %s", p.cfg.Check)
	}
	return nil
}
