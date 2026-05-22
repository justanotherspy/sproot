package modules

// docker installs docker-ce via the official convenience script.
// An optional daemon_json map is written to /etc/docker/daemon.json after install.
//
//	- type: docker
//
//	- type: docker
//	  daemon_json:
//	    log-driver: json-file
//	    log-opts:
//	      max-size: 10m
//	      max-file: "3"

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("docker", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Docker == nil {
			cfg.Docker = &config.DockerConfig{}
		}
		return &dockerPhase{cfg: cfg.Docker}, nil
	})
}

type dockerPhase struct {
	cfg *config.DockerConfig
}

func (p *dockerPhase) Type() string { return "docker" }
func (p *dockerPhase) Name() string { return "docker" }

func (p *dockerPhase) ShouldRun(_ *phase.Context) (bool, error) {
	if !checkCmd("docker", "--version") {
		return true, nil
	}
	if len(p.cfg.DaemonJSON) > 0 {
		if _, err := os.Stat("/etc/docker/daemon.json"); os.IsNotExist(err) {
			return true, nil
		}
	}
	return false, nil
}

func (p *dockerPhase) Run(ctx *phase.Context) error {
	// Install docker-ce via the convenience script from get.docker.com.
	// The script is idempotent; it skips install if docker is already present.
	if err := runCmd(ctx.Log, "sh", "-c",
		"curl -fsSL https://get.docker.com | sh"); err != nil {
		return fmt.Errorf("docker: install script: %w", err)
	}
	if len(p.cfg.DaemonJSON) > 0 {
		if err := writeDaemonJSON(p.cfg.DaemonJSON); err != nil {
			return fmt.Errorf("docker: daemon_json: %w", err)
		}
		if err := runCmd(ctx.Log, "systemctl", "restart", "docker"); err != nil {
			return fmt.Errorf("docker: restart after daemon_json: %w", err)
		}
	}
	return nil
}

func (p *dockerPhase) Verify(_ *phase.Context) error {
	if !checkCmd("docker", "--version") {
		return fmt.Errorf("docker: docker not on PATH after install")
	}
	if len(p.cfg.DaemonJSON) > 0 {
		if _, err := os.Stat("/etc/docker/daemon.json"); err != nil {
			return fmt.Errorf("docker: /etc/docker/daemon.json missing after install")
		}
	}
	return nil
}

func writeDaemonJSON(data map[string]any) error {
	if err := os.MkdirAll("/etc/docker", 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal daemon_json: %w", err)
	}
	return os.WriteFile("/etc/docker/daemon.json", b, 0o644)
}
