package modules

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func newRustComponentsPhase(components []string) *rustComponentsPhase {
	return &rustComponentsPhase{cfg: &config.RustComponentsConfig{Components: components}}
}

func TestRustComponents_ShouldRunWhenRustupMissing(t *testing.T) {
	if _, err := exec.LookPath("rustup"); err != nil {
		p := newRustComponentsPhase([]string{"clippy", "rustfmt"})
		should, err := p.ShouldRun(testCtx(t))
		if err != nil {
			t.Fatal(err)
		}
		if !should {
			t.Error("expected ShouldRun=true when rustup not available")
		}
		return
	}
	t.Skip("rustup on PATH; skip missing-tool path")
}

func TestRustComponents_ShouldRunFalseWhenComponentsInstalled(t *testing.T) {
	if _, err := exec.LookPath("rustup"); err != nil {
		t.Skip("rustup not on PATH")
	}
	components := []string{"clippy", "rustfmt", "rust-analyzer"}
	installed, err := outputOf("rustup", "component", "list", "--installed")
	if err != nil {
		t.Skip("rustup component list failed")
	}
	for _, c := range components {
		if !strings.Contains(installed, c) {
			t.Skipf("component %q not installed; skipping already-done check", c)
		}
	}

	p := newRustComponentsPhase(components)
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when all components already installed")
	}
}

func TestRustComponents_TypeAndName(t *testing.T) {
	p := newRustComponentsPhase(nil)
	if p.Type() != "rust_components" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "rust_components" {
		t.Errorf("Name: got %q", p.Name())
	}
}
