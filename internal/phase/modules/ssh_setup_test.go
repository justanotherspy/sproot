package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHSetup_ShouldRunWhenKeyMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := &sshSetupPhase{ghRegister: stubGHRegister}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when key missing")
	}
}

func TestSSHSetup_RunAndVerify(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not on PATH")
	}
	if _, err := exec.LookPath("ssh-keyscan"); err != nil {
		t.Skip("ssh-keyscan not on PATH")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := &sshSetupPhase{ghRegister: stubGHRegister}
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	sshDir := filepath.Join(home, ".ssh")

	// private and public key must both exist
	if _, err := os.Stat(filepath.Join(sshDir, "id_ed25519")); err != nil {
		t.Error("private key not generated")
	}
	if _, err := os.Stat(filepath.Join(sshDir, "id_ed25519.pub")); err != nil {
		t.Error("public key not generated")
	}

	// known_hosts must contain github.com
	khPath := filepath.Join(sshDir, "known_hosts")
	if !checkCmd("ssh-keygen", "-F", "github.com", "-f", khPath) {
		t.Error("github.com not found in known_hosts")
	}

	// allowed_signers must contain the email
	signers, _ := os.ReadFile(filepath.Join(sshDir, "allowed_signers"))
	if !strings.Contains(string(signers), ctx.Identity.GitUserEmail) {
		t.Error("allowed_signers missing user email")
	}

	if err := p.Verify(ctx); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestSSHSetup_ShouldRunFalseAfterRun(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not on PATH")
	}
	if _, err := exec.LookPath("ssh-keyscan"); err != nil {
		t.Skip("ssh-keyscan not on PATH")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)

	ctx := testCtx(t)
	p := &sshSetupPhase{ghRegister: stubGHRegister}
	if err := p.Run(ctx); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after setup")
	}
}

func TestSSHSetup_SkipsGHRegistrationWithoutToken(t *testing.T) {
	if _, err := exec.LookPath("ssh-keygen"); err != nil {
		t.Skip("ssh-keygen not on PATH")
	}
	if _, err := exec.LookPath("ssh-keyscan"); err != nil {
		t.Skip("ssh-keyscan not on PATH")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GH_TOKEN", "")

	called := false
	p := &sshSetupPhase{ghRegister: func(_, _, _ string) (int64, int64, error) {
		called = true
		return 0, 0, nil
	}}

	if err := p.Run(testCtx(t)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if called {
		t.Error("expected ghRegister not to be called when GH_TOKEN is empty")
	}
}

// stubGHRegister is a no-op GitHub registration stub for tests.
func stubGHRegister(_, _, _ string) (int64, int64, error) { return 1, 2, nil }
