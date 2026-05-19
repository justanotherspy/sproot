package sprite

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newLocalRepo creates a bare-enough git repo in dir with a minimal sproot.yaml
// and returns a file:// URL pointing to it.
func newLocalRepo(t *testing.T, sprootYAML string) string {
	t.Helper()
	dir := t.TempDir()

	// Configure git identity for this repo only.
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

	if err := os.WriteFile(filepath.Join(dir, "sproot.yaml"), []byte(sprootYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	run("add", "sproot.yaml")
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
