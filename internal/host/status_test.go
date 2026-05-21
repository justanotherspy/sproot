package host

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justanotherspy/sproot/internal/phase"
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

func TestRunStatus_HostFlag_ReadsStateFromFilesystem(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	state := phase.State{
		SchemaVersion: 1,
		UpdatedAt:     time.Now(),
		Phases: []phase.PhaseRecord{
			{Type: "apt", Name: "apt", LastRunAt: time.Now(), DidWork: true},
		},
	}
	stateJSON, _ := json.Marshal(state)

	handle := newMockHandle()
	handle.readFiles[spriteStatePath] = stateJSON
	client := &mockClient{handle: handle}

	err := RunStatus(context.Background(), StatusOptions{
		Name:   "my-sprite",
		Host:   true,
		client: client,
	})
	if err != nil {
		t.Fatalf("RunStatus --host: %v", err)
	}
	// --host should NOT exec into the sprite
	if handle.lastCmdName != "" {
		t.Errorf("expected no command execution, got %q", handle.lastCmdName)
	}
}

func TestRunStatus_HostFlag_MissingState(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle() // readFiles empty -> ErrNotExist
	client := &mockClient{handle: handle}

	err := RunStatus(context.Background(), StatusOptions{
		Name:   "my-sprite",
		Host:   true,
		client: client,
	})
	if err == nil || !strings.Contains(err.Error(), "read state") {
		t.Errorf("expected read state error, got %v", err)
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
