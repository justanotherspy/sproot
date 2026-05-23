package modules

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

func completionCtx() *phase.Context {
	return &phase.Context{Log: log.New(io.Discard)}
}

// fakeCompletionTool installs a script named command onto PATH that prints a
// recognizable completion body for `command completion <shell>`, mirroring the
// cobra convention.
func fakeCompletionTool(t *testing.T, command string) {
	t.Helper()
	bin := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1\" = \"completion\" ]; then\n" +
		"  echo \"# completion for " + command + " ($2)\"\n" +
		"  exit 0\n" +
		"fi\n" +
		"exit 1\n"
	if err := os.WriteFile(filepath.Join(bin, command), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func runShellCompletion(t *testing.T, cfg *config.ShellCompletionConfig) *shellCompletionPhase {
	t.Helper()
	p := &shellCompletionPhase{cfg: cfg}
	ctx := completionCtx()

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatalf("ShouldRun: %v", err)
	}
	if !should {
		t.Fatal("expected ShouldRun=true on a fresh home")
	}
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if err := p.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	return p
}

func TestShellCompletion_InstallsAllShells(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fakeCompletionTool(t, "mytool")

	cfg := &config.ShellCompletionConfig{
		Completions: []config.ShellCompletionEntry{
			{Command: "mytool", Shells: []string{"bash", "zsh", "fish"}},
		},
	}
	runShellCompletion(t, cfg)

	wants := map[string]string{
		filepath.Join(home, ".local/share/bash-completion/completions/mytool"): "# completion for mytool (bash)",
		filepath.Join(home, ".zfunc/_mytool"):                                  "# completion for mytool (zsh)",
		filepath.Join(home, ".config/fish/completions/mytool.fish"):            "# completion for mytool (fish)",
	}
	for path, wantSubstr := range wants {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("expected %s to exist: %v", path, err)
			continue
		}
		if !strings.Contains(string(data), wantSubstr) {
			t.Errorf("%s = %q, want substring %q", path, data, wantSubstr)
		}
	}

	zshrc, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	if err != nil {
		t.Fatalf("read .zshrc: %v", err)
	}
	if !strings.Contains(string(zshrc), "fpath=(~/.zfunc") {
		t.Errorf(".zshrc missing fpath block: %q", zshrc)
	}
}

func TestShellCompletion_IdempotentAfterRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	fakeCompletionTool(t, "mytool")

	cfg := &config.ShellCompletionConfig{
		Completions: []config.ShellCompletionEntry{
			{Command: "mytool", Shells: []string{"bash", "zsh"}},
		},
	}
	p := runShellCompletion(t, cfg)

	should, err := p.ShouldRun(completionCtx())
	if err != nil {
		t.Fatalf("ShouldRun (second): %v", err)
	}
	if should {
		t.Error("expected ShouldRun=false after a successful run (idempotent)")
	}
}

func TestShellCompletion_NoZshSkipsRcBlock(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	fakeCompletionTool(t, "mytool")

	cfg := &config.ShellCompletionConfig{
		Completions: []config.ShellCompletionEntry{
			{Command: "mytool", Shells: []string{"bash", "fish"}},
		},
	}
	runShellCompletion(t, cfg)

	if _, err := os.Stat(filepath.Join(home, ".zshrc")); !os.IsNotExist(err) {
		t.Errorf("expected no ~/.zshrc when zsh is not requested, stat err = %v", err)
	}
}

func TestShellCompletion_GenOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	bin := t.TempDir()
	script := "#!/bin/sh\nif [ \"$1\" = \"--emit\" ]; then echo \"custom $2\"; exit 0; fi\nexit 1\n"
	if err := os.WriteFile(filepath.Join(bin, "weird"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))

	cfg := &config.ShellCompletionConfig{
		Completions: []config.ShellCompletionEntry{
			{Command: "weird", Shells: []string{"bash"}, Gen: "{command} --emit {shell}"},
		},
	}
	runShellCompletion(t, cfg)

	data, err := os.ReadFile(filepath.Join(home, ".local/share/bash-completion/completions/weird"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "custom bash") {
		t.Errorf("gen override not honored: %q", data)
	}
}

func TestShellCompletion_GenFailureErrors(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := &config.ShellCompletionConfig{
		Completions: []config.ShellCompletionEntry{
			{Command: "definitely-not-a-real-binary-xyz", Shells: []string{"bash"}},
		},
	}
	p := &shellCompletionPhase{cfg: cfg}
	if err := p.Run(completionCtx()); err == nil {
		t.Fatal("expected Run to fail when the gen command is missing")
	}
}
