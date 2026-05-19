package modules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func newRCBlockPhase(src string) *rcBlockPhase {
	return &rcBlockPhase{cfg: &config.RCBlockConfig{Src: src}}
}

func TestRCBlock_ShouldRunWhenBashrcMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	ctx := testCtx(t)
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte("echo hi\n"), 0o644)

	p := newRCBlockPhase("rc.sh")
	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when .bashrc missing")
	}
}

func TestRCBlock_RunAndVerify(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)
	src := "export FOO=bar\n"
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte(src), 0o644)

	p := newRCBlockPhase("rc.sh")
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, rc := range []string{".bashrc", ".zshrc"} {
		content, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			t.Fatalf("read %s: %v", rc, err)
		}
		s := string(content)
		if !strings.Contains(s, src) {
			t.Errorf("%s: block body missing", rc)
		}
		if !strings.Contains(s, rcBegin) {
			t.Errorf("%s: sentinel begin missing", rc)
		}
	}

	if err := p.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestRCBlock_ShouldRunFalseAfterRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte("export FOO=bar\n"), 0o644)

	p := newRCBlockPhase("rc.sh")
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after block written")
	}
}

func TestRCBlock_ReplacesExistingBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte("export V=1\n"), 0o644)
	p := newRCBlockPhase("rc.sh")
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte("export V=2\n"), 0o644)
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	content, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	s := string(content)
	if strings.Count(s, rcBegin) != 1 {
		t.Errorf("expected exactly one managed block, got:\n%s", s)
	}
	if strings.Contains(s, "V=1") {
		t.Error("old block content still present after replace")
	}
	if !strings.Contains(s, "V=2") {
		t.Error("new block content missing after replace")
	}
}
