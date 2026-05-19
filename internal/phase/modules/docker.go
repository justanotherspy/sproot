package modules

// docker installs docker-ce via the official convenience script.
//
//	- type: docker
//	  docker: {}

import (
	"fmt"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("docker", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &dockerPhase{}, nil
	})
}

type dockerPhase struct{}

func (p *dockerPhase) Type() string { return "docker" }
func (p *dockerPhase) Name() string { return "docker" }

func (p *dockerPhase) ShouldRun(_ *phase.Context) (bool, error) {
	return !checkCmd("docker", "--version"), nil
}

func (p *dockerPhase) Run(ctx *phase.Context) error {
	// Install docker-ce via the convenience script from get.docker.com.
	// Using the script avoids managing apt repo keys manually.
	if err := runCmd(ctx.Log, "sh", "-c",
		"curl -fsSL https://get.docker.com | sh"); err != nil {
		return fmt.Errorf("docker: install script: %w", err)
	}
	return nil
}

func (p *dockerPhase) Verify(_ *phase.Context) error {
	if !checkCmd("docker", "--version") {
		return fmt.Errorf("docker: docker not on PATH after install")
	}
	return nil
}
