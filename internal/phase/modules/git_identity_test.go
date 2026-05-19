package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

// setGlobalGitConfig points git to a temp config file for the duration of the test.
func setGlobalGitConfig(t *testing.T) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), ".gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", tmp)
	return tmp
}

func newGitIdentityPhase(extra map[string]string) *gitIdentityPhase {
	return &gitIdentityPhase{cfg: &config.GitIdentityConfig{Config: extra}}
}

func TestGitIdentity_ShouldRunWhenNotConfigured(t *testing.T) {
	setGlobalGitConfig(t)

	p := newGitIdentityPhase(nil)
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when git not configured")
	}
}

func TestGitIdentity_RunAndVerify(t *testing.T) {
	setGlobalGitConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := newGitIdentityPhase(nil)

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	name, _ := outputOf("git", "config", "--global", "user.name")
	if name != ctx.Identity.GitUserName {
		t.Errorf("user.name: got %q, want %q", name, ctx.Identity.GitUserName)
	}
	email, _ := outputOf("git", "config", "--global", "user.email")
	if email != ctx.Identity.GitUserEmail {
		t.Errorf("user.email: got %q, want %q", email, ctx.Identity.GitUserEmail)
	}

	if err := p.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestGitIdentity_ExtraConfigApplied(t *testing.T) {
	setGlobalGitConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := newGitIdentityPhase(map[string]string{"pull.rebase": "true"})

	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	got, _ := outputOf("git", "config", "--global", "pull.rebase")
	if got != "true" {
		t.Errorf("pull.rebase: got %q, want %q", got, "true")
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after config applied")
	}
}

func TestGitIdentity_ShouldRunFalseAfterRun(t *testing.T) {
	setGlobalGitConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := newGitIdentityPhase(nil)
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	// Signing config is set when pub key exists; add a fake pub key.
	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0o700)
	_ = os.WriteFile(filepath.Join(sshDir, "id_ed25519.pub"), []byte("ssh-ed25519 AAAA fake\n"), 0o644)

	// Re-run to pick up signing config.
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after identity configured")
	}
}

func TestGitIdentity_HandlesSpritePlaceholder(t *testing.T) {
	cfgPath := setGlobalGitConfig(t)
	_ = os.WriteFile(cfgPath, []byte("[user]\n\tname = Sprite\n\temail = noreply@sprites.dev\n"), 0o644)

	p := newGitIdentityPhase(nil)
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when sprite placeholder identity present")
	}
}
