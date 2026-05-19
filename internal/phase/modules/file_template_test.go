package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestFileTemplate_ShouldRunWhenDestMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "file.txt"), []byte("hello\n"), 0o644)
	p := &fileTemplatePhase{cfg: &config.FileTemplateConfig{
		Src:  "file.txt",
		Dest: filepath.Join(home, "out.txt"),
	}}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when dest missing")
	}
}

func TestFileTemplate_RunAndVerify(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	content := "hello world\n"
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "file.txt"), []byte(content), 0o644)
	dest := filepath.Join(home, "out.txt")

	p := &fileTemplatePhase{cfg: &config.FileTemplateConfig{
		Src:  "file.txt",
		Dest: dest,
		Mode: "0644",
	}}

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("read dest: %v", err)
	}
	if string(got) != content {
		t.Errorf("content: got %q, want %q", got, content)
	}

	if err := p.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestFileTemplate_ShouldRunFalseAfterRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "file.txt"), []byte("hi\n"), 0o644)
	dest := filepath.Join(home, "out.txt")

	p := &fileTemplatePhase{cfg: &config.FileTemplateConfig{Src: "file.txt", Dest: dest}}
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after file written")
	}
}

func TestFileTemplate_GoTemplate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	tmpl := "name: {{.GitUserName}}\nemail: {{.GitUserEmail}}\n"
	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "tmpl.txt"), []byte(tmpl), 0o644)
	dest := filepath.Join(home, "rendered.txt")

	p := &fileTemplatePhase{cfg: &config.FileTemplateConfig{Src: "tmpl.txt", Dest: dest}}
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, _ := os.ReadFile(dest)
	want := "name: Test User\nemail: test@example.com\n"
	if string(got) != want {
		t.Errorf("rendered: got %q, want %q", got, want)
	}
}

func TestFileTemplate_TildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	ctx := testCtx(t)

	_ = os.WriteFile(filepath.Join(ctx.ConfigRepoPath, "f.txt"), []byte("x"), 0o644)
	p := &fileTemplatePhase{cfg: &config.FileTemplateConfig{
		Src:  "f.txt",
		Dest: "~/output.txt",
	}}

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, "output.txt")); err != nil {
		t.Fatal("file not created at expanded path")
	}
}
