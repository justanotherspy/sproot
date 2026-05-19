package modules

import (
	"os"
	"path/filepath"
	"testing"
)

// setGlobalGitConfig points git to a temp config file for the duration of the test.
func setGlobalGitConfig(t *testing.T) string {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), ".gitconfig")
	t.Setenv("GIT_CONFIG_GLOBAL", tmp)
	return tmp
}

func TestGitIdentity_ShouldRunWhenNotConfigured(t *testing.T) {
	setGlobalGitConfig(t)

	p := &gitIdentityPhase{}
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
	// ssh key not present, so signing config will be skipped.
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := &gitIdentityPhase{}

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

func TestGitIdentity_ShouldRunFalseAfterRun(t *testing.T) {
	setGlobalGitConfig(t)
	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := &gitIdentityPhase{}
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
	// Pre-populate with the sprite placeholder identity.
	_ = os.WriteFile(cfgPath, []byte("[user]\n\tname = Sprite\n\temail = noreply@sprites.dev\n"), 0o644)

	p := &gitIdentityPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when sprite placeholder identity present")
	}
}
