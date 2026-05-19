package modules

import (
	"os/exec"
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
