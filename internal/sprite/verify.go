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

// phaseForTool maps a command name to the phase type that installs it.
// An empty phaseType means the tool is always checked (e.g. git is a baseline).
var phaseForTool = []struct {
	cmd       string
	phaseType string
}{
	{"git", ""},
	{"gh", "gh_token"},
	{"go", "go_install"},
	{"cargo", "cargo_install"},
	{"uv", "uv_tool"},
	{"docker", "docker"},
	{"node", "corepack"},
	{"pnpm", "corepack"},
	{"rustup", "rust_components"},
}

// phaseRelevant reports whether a check associated with phaseType should run.
// When only is empty (no --only filter) every check runs. When only is set,
// only checks whose phaseType matches the filter (or is empty) are performed.
func phaseRelevant(only, phaseType string) bool {
	if only == "" || phaseType == "" {
		return true
	}
	return only == phaseType
}

// Run checks commands on PATH, SSH key permissions, RC block presence, gh auth,
// and SSH connectivity to github.com. When --only was set (ctx.OnlyFilter non-empty),
// only checks relevant to the filtered phase type are performed so that a partial
// run does not fail on tools that the skipped phases would have installed.
func (v *verifyPhase) Run(ctx *phase.Context) error {
	only := ctx.OnlyFilter
	var errs []error

	for _, tc := range phaseForTool {
		if !phaseRelevant(only, tc.phaseType) {
			continue
		}
		if _, err := exec.LookPath(tc.cmd); err != nil {
			ctx.Log.Errorf("verify: %s not found on PATH", tc.cmd)
			errs = append(errs, fmt.Errorf("%s not on PATH", tc.cmd))
		} else {
			ctx.Log.Successf("verify: %s ok", tc.cmd)
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		if phaseRelevant(only, "ssh_setup") || phaseRelevant(only, "rc_block") {
			ctx.Log.Errorf("verify: cannot resolve home directory: %v", err)
			errs = append(errs, fmt.Errorf("resolve home directory: %w", err))
		}
	} else {
		if phaseRelevant(only, "ssh_setup") {
			errs = append(errs, v.checkSSHKey(ctx, home)...)
		}
		if phaseRelevant(only, "rc_block") {
			errs = append(errs, v.checkRCBlocks(ctx, home)...)
		}
	}

	if phaseRelevant(only, "gh_token") {
		if err := v.checkGHAuth(ctx); err != nil {
			errs = append(errs, err)
		}
	}

	if phaseRelevant(only, "ssh_setup") {
		if err := v.checkSSHGitHub(ctx); err != nil {
			errs = append(errs, err)
		}
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
