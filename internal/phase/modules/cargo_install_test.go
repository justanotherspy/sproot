package modules

import (
	"os/exec"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestCargoIsInstalled(t *testing.T) {
	list := "ripgrep v14.1.0:\n    rg\ncargo-nextest v0.9.72:\n    cargo-nextest\nhexyl v0.14.0:\n    hexyl\n"
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
		// A pinned version must match exactly; a prefix (v0.14.0 vs v0.14) or a
		// superstring (v14.1.0 vs v14.1) must not be considered satisfied.
		{"hexyl", "0.14", false},
		{"ripgrep", "14.1", false},
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

func TestCargoInstall_VerifyUsesBinOverride(t *testing.T) {
	bin := "sh" // reliably on PATH; stands in for a crate's differently-named binary
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("%s not on PATH", bin)
	}
	// Crate name is bogus but bin resolves: Verify must check bin, so it passes.
	pass := &cargoInstallPhase{cfg: &config.CargoInstallConfig{
		Tools: []config.CargoTool{{Name: "sproot-nonexistent-xyzzy-crate", Bin: bin}},
	}}
	if err := pass.Verify(testCtx(t)); err != nil {
		t.Errorf("expected Verify to pass using bin override %q, got: %v", bin, err)
	}
	// Crate name resolves but bin is bogus: Verify must check bin, so it fails.
	fail := &cargoInstallPhase{cfg: &config.CargoInstallConfig{
		Tools: []config.CargoTool{{Name: bin, Bin: "sproot-nonexistent-xyzzy-bin"}},
	}}
	if err := fail.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when bin override not on PATH")
	}
}
