package modules

// go_install installs Go tools via go install pkg@version.
//
//	- type: go_install
//	  tools:
//	    - pkg: golang.org/x/tools/cmd/goimports
//	      version: latest
//	    - pkg: github.com/owner/tool
//	      version: v1.2.3

import (
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("go_install", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.GoInstall == nil {
			return nil, fmt.Errorf("go_install: config is nil")
		}
		return &goInstallPhase{cfg: cfg.GoInstall}, nil
	})
}

type goInstallPhase struct {
	cfg *config.GoInstallConfig
}

func (p *goInstallPhase) Type() string { return "go_install" }
func (p *goInstallPhase) Name() string { return "go_install" }

func (p *goInstallPhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, t := range p.cfg.Tools {
		bin := binaryName(t.Pkg)
		binPath, err := exec.LookPath(bin)
		if err != nil {
			return true, nil
		}
		if t.Version == "latest" {
			// Always re-run for latest to pick up newer releases.
			return true, nil
		}
		// Check installed module path and version match.
		out, err := outputOf("go", "version", "-m", binPath)
		if err != nil {
			return true, nil
		}
		if !strings.Contains(out, t.Pkg) || !strings.Contains(out, t.Version) {
			return true, nil
		}
	}
	return false, nil
}

func (p *goInstallPhase) Run(ctx *phase.Context) error {
	for _, t := range p.cfg.Tools {
		ver := t.Version
		if ver == "" {
			ver = "latest"
		}
		if err := runCmd(ctx.Log, "go", "install", t.Pkg+"@"+ver); err != nil {
			return err
		}
	}
	return nil
}

func (p *goInstallPhase) Verify(_ *phase.Context) error {
	for _, t := range p.cfg.Tools {
		bin := binaryName(t.Pkg)
		if _, err := exec.LookPath(bin); err != nil {
			return fmt.Errorf("go_install: %q not on PATH after install", bin)
		}
	}
	return nil
}

// binaryName returns the last path component of a Go package import path,
// which is conventionally the binary name.
func binaryName(pkg string) string {
	return path.Base(pkg)
}
