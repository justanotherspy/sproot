package modules

import (
	"os/exec"
	"strings"
	"testing"
)

func TestRustComponents_ShouldRunWhenRustupMissing(t *testing.T) {
	if _, err := exec.LookPath("rustup"); err != nil {
		// No rustup: ShouldRun should return true (needs installation).
		p := &rustComponentsPhase{}
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
	installed, err := outputOf("rustup", "component", "list", "--installed")
	if err != nil {
		t.Skip("rustup component list failed")
	}
	allPresent := true
	for _, c := range requiredComponents {
		if !strings.Contains(installed, c) {
			allPresent = false
			break
		}
	}
	if !allPresent {
		t.Skip("not all required components installed; skipping already-done check")
	}

	p := &rustComponentsPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when all components already installed")
	}
}

func TestRustComponents_TypeAndName(t *testing.T) {
	p := &rustComponentsPhase{}
	if p.Type() != "rust_components" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "rust_components" {
		t.Errorf("Name: got %q", p.Name())
	}
}
