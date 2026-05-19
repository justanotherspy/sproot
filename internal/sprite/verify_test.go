package sprite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

func testCtx(t *testing.T) *phase.Context {
	t.Helper()
	return &phase.Context{Log: log.New(os.Stderr)}
}

func TestVerifyPhase_Type(t *testing.T) {
	v := newVerifyPhase()
	if v.Type() != "verify" {
		t.Errorf("Type() = %q, want %q", v.Type(), "verify")
	}
}

func TestVerifyPhase_ShouldRunAlwaysTrue(t *testing.T) {
	v := newVerifyPhase()
	ok, err := v.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatalf("ShouldRun error: %v", err)
	}
	if !ok {
		t.Error("ShouldRun() = false, want true")
	}
}

func TestVerifyPhase_VerifyNoOp(t *testing.T) {
	if err := newVerifyPhase().Verify(testCtx(t)); err != nil {
		t.Errorf("Verify() returned unexpected error: %v", err)
	}
}

func TestVerifyPhase_SSHKeyPermissions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		t.Fatal(err)
	}
	keyPath := filepath.Join(sshDir, "id_ed25519")

	// Write key with wrong permissions.
	if err := os.WriteFile(keyPath, []byte("fake key"), 0o644); err != nil {
		t.Fatal(err)
	}

	v := newVerifyPhase()
	var errs []error
	errs = append(errs, v.checkSSHKey(testCtx(t), home)...)
	if len(errs) == 0 {
		t.Error("expected error for mode 0644, got none")
	}

	// Fix permissions.
	if err := os.Chmod(keyPath, 0o600); err != nil {
		t.Fatal(err)
	}
	errs = errs[:0]
	errs = append(errs, v.checkSSHKey(testCtx(t), home)...)
	if len(errs) != 0 {
		t.Errorf("expected no errors for mode 0600, got: %v", errs)
	}
}

func TestVerifyPhase_RCBlockMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write RC files without the sentinel.
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := os.WriteFile(filepath.Join(home, rc), []byte("# plain rc file\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	v := newVerifyPhase()
	errs := v.checkRCBlocks(testCtx(t), home)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors (one per RC file), got %d: %v", len(errs), errs)
	}
}

func TestVerifyPhase_RCBlockPresent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	content := "# stuff\n" + rcBlockSentinel + "\nsome content\n# END SPROOT MANAGED BLOCK\n"
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := os.WriteFile(filepath.Join(home, rc), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	v := newVerifyPhase()
	errs := v.checkRCBlocks(testCtx(t), home)
	if len(errs) != 0 {
		t.Errorf("expected no errors when sentinel present, got: %v", errs)
	}
}

func TestVerifyPhase_RCBlockFileMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// No RC files written.

	v := newVerifyPhase()
	errs := v.checkRCBlocks(testCtx(t), home)
	if len(errs) != 2 {
		t.Errorf("expected 2 errors for missing RC files, got %d: %v", len(errs), errs)
	}
}
