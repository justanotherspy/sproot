package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConfigInit_WritesGitSkeleton(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sproot", "config")

	if err := RunConfigInit(path, "git"); err != nil {
		t.Fatalf("RunConfigInit: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)

	for _, want := range []string{"config_source: git", "config_repo", "config_ref", "token_env", "gh_token_env"} {
		if !strings.Contains(content, want) {
			t.Errorf("git skeleton missing %q", want)
		}
	}

	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("file mode: got %o, want 0600", info.Mode().Perm())
	}
}

func TestRunConfigInit_WritesLocalSkeleton(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".sproot", "config")

	if err := RunConfigInit(path, "local"); err != nil {
		t.Fatalf("RunConfigInit local: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	content := string(data)

	for _, want := range []string{"config_source: local", "config_local_path", "token_env"} {
		if !strings.Contains(content, want) {
			t.Errorf("local skeleton missing %q", want)
		}
	}
}

func TestRunConfigInit_ErrorIfExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	if err := os.WriteFile(path, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := RunConfigInit(path, "git")
	if err == nil {
		t.Fatal("expected error when file exists, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error %q does not mention 'already exists'", err.Error())
	}
}

func TestRunConfigValidate_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	content := `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: SPRITES_TOKEN
gh_token_env: GITHUB_TOKEN
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RunConfigValidate(path); err != nil {
		t.Fatalf("expected valid config to pass, got: %v", err)
	}
}

func TestRunConfigValidate_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config")

	content := `
config_repo: ""
config_ref: main
token_env: SPRITES_TOKEN
gh_token_env: GITHUB_TOKEN
`
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := RunConfigValidate(path); err == nil {
		t.Fatal("expected validation error for empty config_repo, got nil")
	}
}

func TestRunConfigValidate_FileNotFound(t *testing.T) {
	err := RunConfigValidate("/nonexistent/path/config")
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}
