package modules

// cargo_install installs Rust crates via cargo install.
//
//	- type: cargo_install
//	  tools:
//	    - name: ripgrep
//	    - name: cargo-nextest
//	      version: "0.9.72"   # optional; omit for latest
//	      locked: true         # optional; passes --locked
//	      features: [foo]      # optional

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("cargo_install", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.CargoInstall == nil {
			return nil, fmt.Errorf("cargo_install: config is nil")
		}
		return &cargoInstallPhase{cfg: cfg.CargoInstall}, nil
	})
}

type cargoInstallPhase struct {
	cfg *config.CargoInstallConfig
}

func (p *cargoInstallPhase) Type() string { return "cargo_install" }
func (p *cargoInstallPhase) Name() string { return "cargo_install" }

func (p *cargoInstallPhase) ShouldRun(_ *phase.Context) (bool, error) {
	root, err := userLocalRoot()
	if err != nil {
		// If we cannot resolve the install root, fall back to running.
		return true, nil
	}
	installed, err := cargoInstalledList(root)
	if err != nil {
		// If cargo is not available, we need to run.
		return true, nil
	}
	for _, t := range p.cfg.Tools {
		if !cargoIsInstalled(installed, t) {
			return true, nil
		}
	}
	return false, nil
}

func (p *cargoInstallPhase) Run(ctx *phase.Context) error {
	// Install into ~/.local (binaries land in ~/.local/bin, on the default PATH)
	// rather than cargo's default root (CARGO_HOME/bin), which is not on PATH
	// inside a sprite.
	root, err := userLocalRoot()
	if err != nil {
		return fmt.Errorf("cargo_install: %w", err)
	}
	for _, t := range p.cfg.Tools {
		args := []string{"install", t.Name}
		if t.Version != "" {
			args = append(args, "--version", t.Version)
		}
		if t.Locked {
			args = append(args, "--locked")
		}
		if len(t.Features) > 0 {
			args = append(args, "--features", strings.Join(t.Features, ","))
		}
		args = append(args, "--root", root)
		if err := runCmd(ctx.Log, "cargo", args...); err != nil {
			return err
		}
	}
	return nil
}

func (p *cargoInstallPhase) Verify(_ *phase.Context) error {
	for _, t := range p.cfg.Tools {
		if _, err := exec.LookPath(t.Name); err != nil {
			return fmt.Errorf("cargo_install: %q not on PATH after install", t.Name)
		}
	}
	return nil
}

// cargoInstalledList returns the output of "cargo install --list" scoped to the
// given install root, so it reads the same tracking file (root/.crates2.json)
// that Run writes to via --root.
func cargoInstalledList(root string) (string, error) {
	return outputOf("cargo", "install", "--list", "--root", root)
}

// cargoIsInstalled checks whether the crate is present in cargo install --list output.
// If a version is specified, it checks for "name vversion" prefix. Otherwise checks for name.
func cargoIsInstalled(list string, t config.CargoTool) bool {
	if t.Version != "" {
		return strings.Contains(list, t.Name+" v"+t.Version)
	}
	for _, line := range strings.Split(list, "\n") {
		if strings.HasPrefix(line, t.Name+" v") {
			return true
		}
	}
	return false
}
