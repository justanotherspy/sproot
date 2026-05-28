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

func TestInstalledModVersion(t *testing.T) {
	out := "/root/.local/bin/tool: go1.22.0\n" +
		"\tpath\tgithub.com/owner/tool/cmd/tool\n" +
		"\tmod\tgithub.com/owner/tool\tv1.2.3\th1:abc=\n" +
		"\tdep\tgolang.org/x/sys\tv0.1.0\th1:def=\n"
	if got := installedModVersion(out); got != "v1.2.3" {
		t.Errorf("installedModVersion: got %q, want v1.2.3", got)
	}
	// A pinned v1.2.3 must not be considered satisfied by an installed v1.2.30.
	if installedModVersion(out) == "v1.2.30" {
		t.Error("v1.2.3 should not match v1.2.30")
	}
	if got := installedModVersion("no mod line here\n"); got != "" {
		t.Errorf("installedModVersion(no mod): got %q, want empty", got)
	}
}

func TestResolveGoVersion(t *testing.T) {
	if got := resolveGoVersion(""); got != "latest" {
		t.Errorf("resolveGoVersion(empty): got %q, want latest", got)
	}
	if got := resolveGoVersion("v1.0.0"); got != "v1.0.0" {
		t.Errorf("resolveGoVersion(v1.0.0): got %q, want v1.0.0", got)
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
