package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunStatus_RunsSetupStatusInSprite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunStatus(context.Background(), StatusOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunStatus: %v", err)
	}

	if handle.lastCmdName != "sproot" {
		t.Errorf("command name: got %q, want sproot", handle.lastCmdName)
	}
	if !contains(handle.lastCmdArgs, "--status") {
		t.Errorf("--status flag missing: %v", handle.lastCmdArgs)
	}
}

func TestRunStatus_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunStatus(context.Background(), StatusOptions{Name: "my-sprite"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}
