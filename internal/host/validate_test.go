package host

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidateSprootConfig_ValidFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sproot.yaml")

	content := `
schema_version: 1
identity:
  git_user_name: "Test User"
  git_user_email: test@example.com
  git_default_branch: main
  gh_username: testuser
phases:
  - type: apt
    packages: [curl]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunValidateSprootConfig(path, false); err != nil {
		t.Fatalf("expected valid config to pass, got: %v", err)
	}
}

func TestRunValidateSprootConfig_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sproot.yaml")

	content := `
schema_version: 1
identity:
  git_user_name: ""
  git_user_email: test@example.com
  git_default_branch: main
  gh_username: testuser
phases:
  - type: apt
    packages: [curl]
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	err := RunValidateSprootConfig(path, false)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error %q does not contain 'validation failed'", err.Error())
	}
}

func TestRunValidateSprootConfig_FileNotFound_NonStrict(t *testing.T) {
	// Non-strict mode: missing file is a warning, not an error.
	err := RunValidateSprootConfig("/nonexistent/sproot.yaml", false)
	if err != nil {
		t.Fatalf("expected no error in non-strict mode for missing file, got: %v", err)
	}
}

func TestRunValidateSprootConfig_FileNotFound_Strict(t *testing.T) {
	// Strict mode: missing file is an error.
	err := RunValidateSprootConfig("/nonexistent/sproot.yaml", true)
	if err == nil {
		t.Fatal("expected error in strict mode for missing file, got nil")
	}
}

func TestRunValidateSprootConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sproot.yaml")

	if err := os.WriteFile(path, []byte(`{{{`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunValidateSprootConfig(path, false); err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

const validSprootYAML = `
schema_version: 1
identity:
  git_user_name: "Test User"
  git_user_email: test@example.com
  git_default_branch: main
  gh_username: testuser
phases:
  - type: apt
    packages: [curl]
`

const invalidSprootYAML = `
schema_version: 1
identity:
  git_user_name: ""
  git_user_email: test@example.com
  git_default_branch: main
  gh_username: testuser
phases:
  - type: apt
    packages: [curl]
`

// writeHomeHostConfig writes ~/.sproot/config.yaml under home with the given body.
func writeHomeHostConfig(t *testing.T, home, body string) {
	t.Helper()
	dir := filepath.Join(home, ".sproot")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRunValidate_GitSource_Valid(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initGitRepo(t, repo, validSprootYAML)
	writeHomeHostConfig(t, home, "sproot_config_repo: "+repo+"\nsproot_config_ref: main\ntoken_env: SPRITES_TOKEN\n")

	if err := RunValidate(false); err != nil {
		t.Fatalf("expected valid config resolved from git repo, got: %v", err)
	}
}

func TestRunValidate_GitSource_Invalid(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	repo := t.TempDir()
	initGitRepo(t, repo, invalidSprootYAML)
	writeHomeHostConfig(t, home, "sproot_config_repo: "+repo+"\nsproot_config_ref: main\ntoken_env: SPRITES_TOKEN\n")

	err := RunValidate(false)
	if err == nil {
		t.Fatal("expected validation error for invalid config in repo, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error %q does not contain 'validation failed'", err.Error())
	}
}

func TestRunValidate_LocalSource_Valid(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	localDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(localDir, "sproot.yaml"), []byte(validSprootYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	writeHomeHostConfig(t, home, "sproot_config_source: local\nsproot_config_local_path: "+localDir+"\ntoken_env: SPRITES_TOKEN\n")

	if err := RunValidate(false); err != nil {
		t.Fatalf("expected valid config resolved from local dir, got: %v", err)
	}
}

func TestRunValidate_NoHostConfig_FallsBackToCwd(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home) // no ~/.sproot/config.yaml here
	cwd := t.TempDir()
	t.Chdir(cwd)

	// No ./sproot.yaml: non-strict is a warning (nil), strict is an error.
	if err := RunValidate(false); err != nil {
		t.Fatalf("non-strict missing file should not error, got: %v", err)
	}
	if err := RunValidate(true); err == nil {
		t.Fatal("strict missing file should error, got nil")
	}

	// With a valid ./sproot.yaml present, it is validated.
	if err := os.WriteFile(filepath.Join(cwd, "sproot.yaml"), []byte(validSprootYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RunValidate(false); err != nil {
		t.Fatalf("expected valid cwd sproot.yaml to pass, got: %v", err)
	}
}
