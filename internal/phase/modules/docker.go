package modules

// docker installs docker-ce via the official convenience script.
// An optional daemon_json map is deep-merged into /etc/docker/daemon.json after
// install (existing keys preserved; lists/scalars replaced wholesale).
//
//	- type: docker
//
//	- type: docker
//	  daemon_json:
//	    storage-driver: overlay2
//	    log-driver: json-file
//	    log-opts:
//	      max-size: 10m
//	      max-file: "3"
//
// On a sprite (sprite-env present) the daemon is not restarted via systemd;
// the merged config is picked up when a later sprite_service phase starts dockerd.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const daemonJSONPath = "/etc/docker/daemon.json"

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
		changed, err := daemonJSONChanged(daemonJSONPath, p.cfg.DaemonJSON)
		if err != nil {
			return false, err
		}
		if changed {
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
		if err := mergeDaemonJSONAt(daemonJSONPath, p.cfg.DaemonJSON); err != nil {
			return fmt.Errorf("docker: daemon_json: %w", err)
		}
		ctx.Log.Infof("merged %s", daemonJSONPath)
		// Sprites have no systemd; a later sprite_service phase starts dockerd
		// with the merged config, so only restart on a real systemd host.
		if spriteEnvPresent() {
			ctx.Log.Infof("sprite-env detected; skipping systemctl restart (sprite_service starts dockerd)")
		} else if err := runCmd(ctx.Log, "systemctl", "restart", "docker"); err != nil {
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
		if _, err := os.Stat(daemonJSONPath); err != nil {
			return fmt.Errorf("docker: %s missing after install", daemonJSONPath)
		}
	}
	return nil
}

// spriteEnvPresent reports whether the sprite-env daemon control tool is available.
func spriteEnvPresent() bool {
	_, err := exec.LookPath("sprite-env")
	return err == nil
}

// mergedDaemonJSON returns the pretty-printed result of deep-merging data into
// any existing daemon.json at path. A missing file starts from an empty object;
// a corrupt existing file is a hard error.
func mergedDaemonJSON(path string, data map[string]any) ([]byte, error) {
	merged := map[string]any{}
	raw, err := os.ReadFile(path)
	switch {
	case err == nil:
		if uerr := json.Unmarshal(raw, &merged); uerr != nil {
			return nil, fmt.Errorf("parse existing %s: %w", path, uerr)
		}
	case errors.Is(err, fs.ErrNotExist):
		// no existing file; start empty
	default:
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	deepMerge(merged, data)
	b, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal daemon_json: %w", err)
	}
	return append(b, '\n'), nil
}

// mergeDaemonJSONAt writes the merged daemon.json to path.
func mergeDaemonJSONAt(path string, data map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := mergedDaemonJSON(path, data)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// daemonJSONChanged reports whether merging data would change the file at path.
func daemonJSONChanged(path string, data map[string]any) (bool, error) {
	want, err := mergedDaemonJSON(path, data)
	if err != nil {
		return false, err
	}
	have, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	return !bytes.Equal(have, want), nil
}
