package modules

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSSHSetup_ShouldRunWhenKnownHostsMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := &sshSetupPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when known_hosts missing")
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

	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0o700)

	// Generate a real ed25519 key so the module has something to work with.
	keyPath := filepath.Join(sshDir, "id_ed25519")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	ctx := testCtx(t)
	p := &sshSetupPhase{}
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// known_hosts must contain github.com.
	kh, err := os.ReadFile(filepath.Join(sshDir, "known_hosts"))
	if err != nil {
		t.Fatal("known_hosts not written")
	}
	if !strings.Contains(string(kh), "github.com") {
		t.Error("known_hosts does not contain github.com")
	}

	// pub key must exist.
	if _, err := os.Stat(keyPath + ".pub"); err != nil {
		t.Error("pubkey not derived")
	}

	// allowed_signers must contain the email.
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

	sshDir := filepath.Join(home, ".ssh")
	_ = os.MkdirAll(sshDir, 0o700)
	keyPath := filepath.Join(sshDir, "id_ed25519")
	if err := exec.Command("ssh-keygen", "-t", "ed25519", "-f", keyPath, "-N", "").Run(); err != nil {
		t.Fatalf("generate test key: %v", err)
	}

	ctx := testCtx(t)
	p := &sshSetupPhase{}
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
