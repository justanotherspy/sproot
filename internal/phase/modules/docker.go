package modules

// docker installs docker-ce via the official apt repository and writes
// a baseline /etc/docker/daemon.json (userland-proxy disabled).
//
//	- type: docker

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const dockerDaemonPath = "/etc/docker/daemon.json"

var dockerDaemonConfig = map[string]any{
	"userland-proxy": false,
}

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
	// Using the script avoids having to manage apt repo keys manually.
	if err := runCmd(ctx.Log, "sh", "-c",
		"curl -fsSL https://get.docker.com | sh"); err != nil {
		return fmt.Errorf("docker: install script: %w", err)
	}
	return p.writeDaemonConfig(ctx)
}

func (p *dockerPhase) writeDaemonConfig(ctx *phase.Context) error {
	data, err := json.MarshalIndent(dockerDaemonConfig, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(dockerDaemonPath, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("docker: write daemon.json: %w", err)
	}
	ctx.Log.Infof("wrote %s", dockerDaemonPath)
	return nil
}

func (p *dockerPhase) Verify(_ *phase.Context) error {
	if !checkCmd("docker", "--version") {
		return fmt.Errorf("docker: docker not on PATH after install")
	}
	if _, err := os.Stat(dockerDaemonPath); err != nil {
		return fmt.Errorf("docker: daemon.json missing at %s", dockerDaemonPath)
	}
	return nil
}
