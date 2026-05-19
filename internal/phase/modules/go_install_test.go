package modules

import (
	"os/exec"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestBinaryName(t *testing.T) {
	cases := []struct {
		pkg  string
		want string
	}{
		{"github.com/owner/tool", "tool"},
		{"golang.org/x/tools/cmd/goimports", "goimports"},
		{"singlesegment", "singlesegment"},
	}
	for _, tc := range cases {
		got := binaryName(tc.pkg)
		if got != tc.want {
			t.Errorf("binaryName(%q): got %q, want %q", tc.pkg, got, tc.want)
		}
	}
}

func TestGoInstall_ShouldRunWhenBinaryMissing(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	p := &goInstallPhase{cfg: &config.GoInstallConfig{
		Tools: []config.GoTool{{Pkg: "github.com/sproot-nonexistent/xyzzy", Version: "v1.0.0"}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true for missing binary")
	}
}

func TestGoInstall_ShouldRunTrueForLatest(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	// version=latest always re-runs, even if the binary exists.
	p := &goInstallPhase{cfg: &config.GoInstallConfig{
		Tools: []config.GoTool{{Pkg: "github.com/sproot-nonexistent/xyzzy", Version: "latest"}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true for version=latest")
	}
}
