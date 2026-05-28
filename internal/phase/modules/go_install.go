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
		if resolveGoVersion(t.Version) == "latest" {
			// Always re-run for latest (the default) to pick up newer releases.
			return true, nil
		}
		// Check installed module path and version match exactly.
		out, err := outputOf("go", "version", "-m", binPath)
		if err != nil {
			return true, nil
		}
		if !strings.Contains(out, t.Pkg) || installedModVersion(out) != t.Version {
			return true, nil
		}
	}
	return false, nil
}

// resolveGoVersion normalizes an empty version to "latest", so ShouldRun and Run
// agree on the meaning of an unset version.
func resolveGoVersion(v string) string {
	if v == "" {
		return "latest"
	}
	return v
}

// installedModVersion extracts the module version from `go version -m` output.
// The relevant line is "\tmod\t<module>\t<version>\t<hash>"; an empty string is
// returned when no mod line is present.
func installedModVersion(out string) string {
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == "mod" {
			return fields[2]
		}
	}
	return ""
}

func (p *goInstallPhase) Run(ctx *phase.Context) error {
	// Install into ~/.local/bin (on the default PATH) rather than the default
	// GOPATH/bin, which is not on PATH inside a sprite.
	bin, err := userLocalBin()
	if err != nil {
		return fmt.Errorf("go_install: %w", err)
	}
	for _, t := range p.cfg.Tools {
		ver := resolveGoVersion(t.Version)
		if err := runCmdEnv([]string{"GOBIN=" + bin}, ctx.Log, "go", "install", t.Pkg+"@"+ver); err != nil {
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
