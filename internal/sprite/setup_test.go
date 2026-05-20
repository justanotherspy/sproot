package sprite

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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
