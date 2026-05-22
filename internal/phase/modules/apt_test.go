package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestApt_ShouldRunWhenPackageMissing(t *testing.T) {
	if _, err := exec.LookPath("dpkg"); err != nil {
		t.Skip("dpkg not on PATH")
	}
	// Use a package name that almost certainly does not exist.
	p := &aptPhase{cfg: &config.AptConfig{Packages: []string{"sproot-nonexistent-pkg-xyzzy"}}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true for missing package")
	}
}

func TestApt_ShouldRunFalseWhenInstalled(t *testing.T) {
	if _, err := exec.LookPath("dpkg"); err != nil {
		t.Skip("dpkg not on PATH")
	}
	// bash is always installed on this sprite.
	p := &aptPhase{cfg: &config.AptConfig{Packages: []string{"bash"}}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false for already-installed package")
	}
}

func TestApt_VerifyFailsWhenPackageMissing(t *testing.T) {
	if _, err := exec.LookPath("dpkg"); err != nil {
		t.Skip("dpkg not on PATH")
	}
	p := &aptPhase{cfg: &config.AptConfig{Packages: []string{"sproot-nonexistent-pkg-xyzzy"}}}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail for missing package")
	}
}

func TestApt_ShouldRunWhenSymlinkMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent-symlink")
	p := &aptPhase{cfg: &config.AptConfig{
		Symlinks: []config.AptSymlink{{From: "/bin/sh", To: missing}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when symlink target does not exist")
	}
}

func TestApt_ShouldRunFalseWhenSymlinkExists(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "mylink")
	if err := os.Symlink("/bin/sh", target); err != nil {
		t.Fatal(err)
	}
	p := &aptPhase{cfg: &config.AptConfig{
		Symlinks: []config.AptSymlink{{From: "/bin/sh", To: target}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when symlink already exists")
	}
}

func TestApt_VerifyFailsWhenSymlinkMissing(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "nonexistent")
	p := &aptPhase{cfg: &config.AptConfig{
		Symlinks: []config.AptSymlink{{From: "/bin/sh", To: missing}},
	}}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when symlink target is missing")
	}
}
