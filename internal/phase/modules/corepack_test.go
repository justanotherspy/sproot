package modules

import (
	"os/exec"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func newCorepackPhase(managers []string) *corepackPhase {
	return &corepackPhase{cfg: &config.CorepackConfig{Managers: managers}}
}

func TestCorepack_ShouldRunWhenManagersMissing(t *testing.T) {
	if _, err := exec.LookPath("corepack"); err != nil {
		t.Skip("corepack not on PATH")
	}
	p := newCorepackPhase([]string{"pnpm", "yarn"})
	_, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatalf("ShouldRun: %v", err)
	}
}

func TestCorepack_VerifyFailsWhenBinariesMissing(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	p := newCorepackPhase([]string{"pnpm", "yarn"})
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when managers not on PATH")
	}
}

func TestCorepack_TypeAndName(t *testing.T) {
	p := newCorepackPhase(nil)
	if p.Type() != "corepack" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "corepack" {
		t.Errorf("Name: got %q", p.Name())
	}
}
