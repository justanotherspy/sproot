package sprite

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/phase"
)

// rcBlockSentinel is the opening line of the sproot-managed RC block written by
// the rc_block module (mirrors rcBegin in internal/phase/modules/rc_block.go).
const rcBlockSentinel = "# BEGIN SPROOT MANAGED BLOCK"

// verifyPhase is the built-in end-of-run verification step. It is appended by
// setup.go and is not user-configurable via sproot.yaml.
type verifyPhase struct{}

func newVerifyPhase() *verifyPhase { return &verifyPhase{} }

func (v *verifyPhase) Type() string                              { return "verify" }
func (v *verifyPhase) Name() string                              { return "verify" }
func (v *verifyPhase) ShouldRun(_ *phase.Context) (bool, error) { return true, nil }
func (v *verifyPhase) Verify(_ *phase.Context) error            { return nil }

// Run checks commands on PATH, SSH key permissions, RC block presence, gh auth,
// and SSH connectivity to github.com. All checks are attempted; failures are
// collected and returned together.
func (v *verifyPhase) Run(ctx *phase.Context) error {
	var errs []error

	for _, cmd := range []string{"git", "gh", "go", "cargo", "uv", "docker", "node", "pnpm", "rustup"} {
		if _, err := exec.LookPath(cmd); err != nil {
			ctx.Log.Errorf("verify: %s not found on PATH", cmd)
			errs = append(errs, fmt.Errorf("%s not on PATH", cmd))
		} else {
			ctx.Log.Successf("verify: %s ok", cmd)
		}
	}

	home, err := os.UserHomeDir()
	if err == nil {
		errs = append(errs, v.checkSSHKey(ctx, home)...)
		errs = append(errs, v.checkRCBlocks(ctx, home)...)
	}

	if err := v.checkGHAuth(ctx); err != nil {
		errs = append(errs, err)
	}

	if err := v.checkSSHGitHub(ctx); err != nil {
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

func (v *verifyPhase) checkSSHKey(ctx *phase.Context, home string) []error {
	keyPath := filepath.Join(home, ".ssh", "id_ed25519")
	info, err := os.Stat(keyPath)
	if err != nil {
		// Key may not be present in all configurations.
		return nil
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		ctx.Log.Errorf("verify: %s has mode %04o, want 0600", keyPath, perm)
		return []error{fmt.Errorf("ssh key %s: mode is %04o, want 0600", keyPath, perm)}
	}
	ctx.Log.Successf("verify: ssh key permissions ok")
	return nil
}

func (v *verifyPhase) checkRCBlocks(ctx *phase.Context, home string) []error {
	var errs []error
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		content, err := os.ReadFile(path)
		if err != nil {
			ctx.Log.Errorf("verify: cannot read %s: %v", rc, err)
			errs = append(errs, fmt.Errorf("cannot read %s: %w", rc, err))
			continue
		}
		if !strings.Contains(string(content), rcBlockSentinel) {
			ctx.Log.Errorf("verify: sproot block missing from %s", rc)
			errs = append(errs, fmt.Errorf("sproot block missing from %s", rc))
		} else {
			ctx.Log.Successf("verify: %s block ok", rc)
		}
	}
	return errs
}

func (v *verifyPhase) checkGHAuth(ctx *phase.Context) error {
	cmd := exec.Command("gh", "auth", "status")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		ctx.Log.Errorf("verify: gh auth status failed")
		return fmt.Errorf("gh auth status: %w", err)
	}
	ctx.Log.Successf("verify: gh auth ok")
	return nil
}

func (v *verifyPhase) checkSSHGitHub(ctx *phase.Context) error {
	cmd := exec.Command("ssh", "-T",
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		"-o", "ConnectTimeout=10",
		"git@github.com",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err == nil {
		ctx.Log.Successf("verify: ssh github.com ok")
		return nil
	}
	// GitHub exits 1 on successful auth (no shell access granted).
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
		ctx.Log.Successf("verify: ssh github.com ok")
		return nil
	}
	ctx.Log.Errorf("verify: ssh github.com failed: %v", err)
	return fmt.Errorf("ssh github.com: %w", err)
}
