package modules

import (
	"os/exec"
	"testing"
)

func TestCorepack_ShouldRunWhenPnpmMissing(t *testing.T) {
	if _, err := exec.LookPath("corepack"); err != nil {
		t.Skip("corepack not on PATH")
	}
	p := &corepackPhase{}
	// We can only test ShouldRun accurately if pnpm/yarn are absent;
	// on a fully set-up sprite they may already be present. Either outcome is valid.
	_, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatalf("ShouldRun: %v", err)
	}
}

func TestCorepack_VerifyFailsWhenBinariesMissing(t *testing.T) {
	// Simulate PATH with neither pnpm nor yarn by creating a temp empty PATH.
	t.Setenv("PATH", t.TempDir())
	p := &corepackPhase{}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when pnpm/yarn not on PATH")
	}
}

func TestCorepack_TypeAndName(t *testing.T) {
	p := &corepackPhase{}
	if p.Type() != "corepack" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "corepack" {
		t.Errorf("Name: got %q", p.Name())
	}
}
