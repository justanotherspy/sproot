package sprite

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/pkg/log"
)

func newTestLogger() *log.Logger { return log.New(io.Discard) }

// newLocalRepo creates a bare-enough git repo with sproot.yaml at the root
// and returns a file:// URL pointing to it.
func newLocalRepo(t *testing.T, sprootYAML string) string {
	t.Helper()
	return newLocalRepoAt(t, "sproot.yaml", sprootYAML)
}

// newLocalRepoAt creates a git repo with the config file at relPath
// and returns a file:// URL pointing to it.
func newLocalRepoAt(t *testing.T, relPath, content string) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}

	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	run("config", "commit.gpgsign", "false")

	fullPath := filepath.Join(dir, filepath.FromSlash(relPath))
	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	run("add", ".")
	run("commit", "-m", "init")

	return "file://" + dir
}

const minimalSprootYAML = `schema_version: 1
identity:
  git_user_name: "Test User"
  git_user_email: "test@example.com"
  git_default_branch: main
  gh_username: testuser
phases:
  - type: cmd
    run: "true"
    check: "true"
`

func TestRunSetup_DryRun(t *testing.T) {
	repoURL := newLocalRepo(t, minimalSprootYAML)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	err := RunSetup(SetupOptions{
		ConfigRepo: repoURL,
		Ref:        "main",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("RunSetup dry-run: %v", err)
	}
}

func TestRunSetup_MissingConfigRepo(t *testing.T) {
	err := RunSetup(SetupOptions{Ref: "main"})
	if err == nil {
		t.Fatal("expected error for missing --config-repo")
	}
}

func TestRunSetup_Status_NoStateFile(t *testing.T) {
	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	// Status with no state file should succeed (prints "no phase records" message).
	err := RunSetup(SetupOptions{Status: true})
	if err != nil {
		t.Fatalf("RunSetup --status with no state file: %v", err)
	}
}

func TestRunSetup_CustomConfigPath(t *testing.T) {
	repoURL := newLocalRepoAt(t, "configs/dev.yaml", minimalSprootYAML)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	err := RunSetup(SetupOptions{
		ConfigRepo: repoURL,
		Ref:        "main",
		ConfigPath: "configs/dev.yaml",
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("RunSetup with custom config path: %v", err)
	}
}

func TestRunSetup_CustomConfigPath_NotFound(t *testing.T) {
	repoURL := newLocalRepo(t, minimalSprootYAML)

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	err := RunSetup(SetupOptions{
		ConfigRepo: repoURL,
		Ref:        "main",
		ConfigPath: "nonexistent/path.yaml",
		DryRun:     true,
	})
	if err == nil {
		t.Fatal("expected error for missing config file")
	}
	if !strings.Contains(err.Error(), "nonexistent/path.yaml") {
		t.Errorf("error should mention the config path, got: %v", err)
	}
}

// TestCloneOrPull_UpdatesRemoteURL covers the 9f fix: if the recorded remote URL
// differs from the requested one, cloneOrPull updates it before fetching.
func TestCloneOrPull_UpdatesRemoteURL(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}

	// Create the original upstream repo.
	origURL := newLocalRepo(t, minimalSprootYAML)

	dest := t.TempDir()
	l := newTestLogger()

	// Initial clone.
	if err := cloneOrPull(l, origURL, "main", dest); err != nil {
		t.Fatalf("initial clone: %v", err)
	}

	// Create a second repo to serve as the new origin.
	newURL := newLocalRepo(t, minimalSprootYAML)

	// cloneOrPull with the new URL: should update origin and fetch successfully.
	if err := cloneOrPull(l, newURL, "main", dest); err != nil {
		t.Fatalf("cloneOrPull after URL change: %v", err)
	}

	// Confirm the recorded remote is now the new URL.
	out, err := gitOutput("-C", dest, "remote", "get-url", "origin")
	if err != nil {
		t.Fatalf("git remote get-url: %v", err)
	}
	if strings.TrimSpace(out) != newURL {
		t.Errorf("remote origin: got %q, want %q", strings.TrimSpace(out), newURL)
	}
}

func TestRewriteConfigRepoURL(t *testing.T) {
	l := newTestLogger()
	if got := rewriteConfigRepoURL(l, "git@github.com:owner/repo.git"); got != "https://github.com/owner/repo.git" {
		t.Errorf("ssh url not rewritten: got %q", got)
	}
	if got := rewriteConfigRepoURL(l, "https://github.com/owner/repo.git"); got != "https://github.com/owner/repo.git" {
		t.Errorf("https url should be unchanged: got %q", got)
	}
	if got := rewriteConfigRepoURL(l, "file:///tmp/repo"); got != "file:///tmp/repo" {
		t.Errorf("local url should be unchanged: got %q", got)
	}
}

func TestRedactGitArgs(t *testing.T) {
	args := []string{
		"-c", "http.https://github.com/.extraheader=AUTHORIZATION: basic eC1hY2Nlc3MtdG9rZW46czNjcmV0",
		"clone", "--branch", "main", "https://github.com/owner/repo.git", "/dest",
	}
	got := redactGitArgs(args)
	if strings.Contains(got, "eC1hY2Nlc3MtdG9rZW46czNjcmV0") {
		t.Errorf("credential not redacted: %q", got)
	}
	if !strings.Contains(got, "basic REDACTED") {
		t.Errorf("expected redacted header, got %q", got)
	}
	if !strings.Contains(got, "clone --branch main") {
		t.Errorf("non-credential args should be preserved, got %q", got)
	}
}

func TestRunSetup_SkipVerify(t *testing.T) {
	// Create a minimal local config dir (no rc_block, so verify will always fail the
	// sentinel check when the phase runs against a fresh HOME).
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "sproot.yaml"), []byte(minimalSprootYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	// Create empty shell RC files so verify can read them but finds no sentinel.
	for _, rc := range []string{".bashrc", ".zshrc"} {
		if err := os.WriteFile(filepath.Join(tmpHome, rc), []byte(""), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Without SkipVerify the built-in verify phase runs and fails (sentinel missing,
	// gh auth unavailable, etc.).
	err := RunSetup(SetupOptions{LocalConfig: dir, SkipVerify: false})
	if err == nil {
		t.Fatal("expected verify phase to fail without SkipVerify=true")
	}

	// With SkipVerify the verify phase is not appended; setup should succeed.
	err = RunSetup(SetupOptions{LocalConfig: dir, SkipVerify: true})
	if err != nil {
		t.Fatalf("RunSetup with SkipVerify=true failed: %v", err)
	}
}

func TestRunSetup_InvalidYAML(t *testing.T) {
	// Repo with invalid sproot.yaml (missing required fields).
	repoURL := newLocalRepo(t, "schema_version: 1\n")

	tmpHome := t.TempDir()
	t.Setenv("HOME", tmpHome)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(tmpHome, ".config"))

	err := RunSetup(SetupOptions{
		ConfigRepo: repoURL,
		Ref:        "main",
		DryRun:     true,
	})
	if err == nil {
		t.Fatal("expected validation error for minimal-only sproot.yaml")
	}
}
