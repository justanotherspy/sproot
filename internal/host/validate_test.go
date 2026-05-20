package host

import (
	"os"
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

	if err := RunValidateSprootConfig(path); err != nil {
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

	err := RunValidateSprootConfig(path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "validation failed") {
		t.Errorf("error %q does not contain 'validation failed'", err.Error())
	}
}

func TestRunValidateSprootConfig_FileNotFound(t *testing.T) {
	err := RunValidateSprootConfig("/nonexistent/sproot.yaml")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestRunValidateSprootConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sproot.yaml")

	if err := os.WriteFile(path, []byte(`{{{`), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RunValidateSprootConfig(path); err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}
