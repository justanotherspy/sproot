package modules

import (
	"os/exec"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestCargoIsInstalled(t *testing.T) {
	list := "ripgrep v14.1.0:\n    rg\ncargo-nextest v0.9.72:\n    cargo-nextest\n"
	cases := []struct {
		name    string
		version string
		want    bool
	}{
		{"ripgrep", "", true},
		{"ripgrep", "14.1.0", true},
		{"ripgrep", "14.0.0", false},
		{"cargo-nextest", "0.9.72", true},
		{"cargo-nextest", "", true},
		{"nonexistent", "", false},
	}
	for _, tc := range cases {
		tool := config.CargoTool{Name: tc.name, Version: tc.version}
		got := cargoIsInstalled(list, tool)
		if got != tc.want {
			t.Errorf("cargoIsInstalled(name=%q, version=%q): got %v, want %v",
				tc.name, tc.version, got, tc.want)
		}
	}
}

func TestCargoInstall_ShouldRunWhenCargoMissing(t *testing.T) {
	if _, err := exec.LookPath("cargo"); err != nil {
		t.Skip("cargo not on PATH")
	}
	// A crate that almost certainly isn't installed.
	p := &cargoInstallPhase{cfg: &config.CargoInstallConfig{
		Tools: []config.CargoTool{{Name: "sproot-nonexistent-xyzzy-crate"}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true for crate that is not installed")
	}
}

func TestCargoInstall_VerifyFailsWhenBinaryMissing(t *testing.T) {
	p := &cargoInstallPhase{cfg: &config.CargoInstallConfig{
		Tools: []config.CargoTool{{Name: "sproot-nonexistent-xyzzy-crate"}},
	}}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when binary not on PATH")
	}
}
