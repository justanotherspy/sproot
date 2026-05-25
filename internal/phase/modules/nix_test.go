package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestNix_TypeAndName(t *testing.T) {
	p := &nixPhase{cfg: &config.NixConfig{}}
	if p.Type() != "nix" {
		t.Errorf("Type() = %q, want nix", p.Type())
	}
	if p.Name() != "nix" {
		t.Errorf("Name() = %q, want nix", p.Name())
	}
}

func TestNixPackage_FlakeRefAndBinName(t *testing.T) {
	tests := []struct {
		pkg       config.NixPackage
		wantFlake string
		wantBin   string
	}{
		{config.NixPackage{Name: "hello"}, "nixpkgs#hello", "hello"},
		{config.NixPackage{Name: "ripgrep", Bin: "rg"}, "nixpkgs#ripgrep", "rg"},
		{config.NixPackage{Flake: "nixpkgs#nixfmt-rfc-style", Bin: "nixfmt"}, "nixpkgs#nixfmt-rfc-style", "nixfmt"},
		{config.NixPackage{Name: "cowsay", Flake: "github:owner/repo#cowsay"}, "github:owner/repo#cowsay", "cowsay"},
	}
	for _, tt := range tests {
		if got := tt.pkg.FlakeRef(); got != tt.wantFlake {
			t.Errorf("FlakeRef() = %q, want %q", got, tt.wantFlake)
		}
		if got := tt.pkg.BinName(); got != tt.wantBin {
			t.Errorf("BinName() = %q, want %q", got, tt.wantBin)
		}
	}
}

func TestNixConfig_DaemonServiceEnabled(t *testing.T) {
	enabled := true
	disabled := false
	cases := []struct {
		cfg  config.NixConfig
		want bool
	}{
		{config.NixConfig{}, true},
		{config.NixConfig{DaemonService: &enabled}, true},
		{config.NixConfig{DaemonService: &disabled}, false},
	}
	for _, c := range cases {
		if got := c.cfg.DaemonServiceEnabled(); got != c.want {
			t.Errorf("DaemonServiceEnabled() = %v, want %v", got, c.want)
		}
	}
}

func TestShellQuote(t *testing.T) {
	cases := map[string]string{
		"nixpkgs#hello": `'nixpkgs#hello'`,
		"a b":           `'a b'`,
		"it's":          `'it'\''s'`,
	}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Errorf("shellQuote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNix_ShouldRunWhenNotInstalled(t *testing.T) {
	// nix is not installed on the test host, so ShouldRun must be true.
	p := &nixPhase{cfg: &config.NixConfig{Packages: []config.NixPackage{{Name: "hello"}}}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when nix is not installed")
	}
}

func TestNix_WriteRCBlockIsIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Seed rc files with prior content to confirm the block appends cleanly.
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := os.WriteFile(filepath.Join(home, rc), []byte("export EXISTING=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	p := &nixPhase{cfg: &config.NixConfig{}}
	ctx := testCtx(t)

	present, err := nixRCPresent()
	if err != nil {
		t.Fatal(err)
	}
	if present {
		t.Fatal("expected block absent before write")
	}

	if err := p.writeRCBlock(ctx); err != nil {
		t.Fatalf("writeRCBlock: %v", err)
	}
	present, err = nixRCPresent()
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("expected block present after write")
	}

	// A second write must not duplicate the block.
	if err := p.writeRCBlock(ctx); err != nil {
		t.Fatalf("writeRCBlock (2nd): %v", err)
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		data, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			t.Fatal(err)
		}
		if n := countOccurrences(string(data), nixRCBegin); n != 1 {
			t.Errorf("%s: expected 1 nix block, got %d", rc, n)
		}
		if !contains(string(data), "export EXISTING=1") {
			t.Errorf("%s: prior content was clobbered", rc)
		}
	}
}

func countOccurrences(s, sub string) int {
	count := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			count++
		}
	}
	return count
}

func contains(s, sub string) bool { return countOccurrences(s, sub) > 0 }
