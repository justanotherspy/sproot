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

// TestRCBlock_ShouldRunWhenOnlyZshrcStale covers the 9a bug: ShouldRun previously
// checked only .bashrc, so a stale .zshrc would not trigger a re-run.
func TestRCBlock_ShouldRunWhenOnlyZshrcStale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)
	src := "export FOO=bar\n"
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte(src), 0o644)

	p := newRCBlockPhase("rc.sh")
	// Write the block only to .bashrc; leave .zshrc absent.
	if err := applyManagedBlock(filepath.Join(home, ".bashrc"), src, rcBegin, rcEnd); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when .zshrc is missing even though .bashrc is current")
	}
}

// TestRCBlock_TrailingNewlineNormalized covers the 9b bug: if src has no trailing
// newline the end sentinel was appended on the same line as the last src line.
func TestRCBlock_TrailingNewlineNormalized(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)
	// Deliberate: no trailing newline.
	src := "export BAR=baz"
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte(src), 0o644)

	p := newRCBlockPhase("rc.sh")
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	content, _ := os.ReadFile(filepath.Join(home, ".bashrc"))
	s := string(content)
	// The end sentinel must appear on its own line, not concatenated with src.
	if strings.Contains(s, src+rcEnd) {
		t.Errorf("end sentinel not on its own line:\n%s", s)
	}
	if !strings.Contains(s, "\n"+rcEnd) {
		t.Errorf("end sentinel missing newline prefix:\n%s", s)
	}
}

// TestRCBlock_IdempotentWithoutTrailingNewline guards against hashing the raw
// src while Run writes a newline-normalized body: ShouldRun and Verify must hash
// the same normalized body, so a source file with no trailing newline is still
// recognized as already applied (and not re-run forever).
func TestRCBlock_IdempotentWithoutTrailingNewline(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)
	// Deliberate: no trailing newline.
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "rc.sh"), []byte("export BAR=baz"), 0o644)

	p := newRCBlockPhase("rc.sh")
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after run even when src lacks a trailing newline")
	}
	if err := p.Verify(ctx); err != nil {
		t.Errorf("Verify should pass after run even when src lacks a trailing newline: %v", err)
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
