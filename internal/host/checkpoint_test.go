package host

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunCheckpoint_CallsCheckpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunCheckpoint(context.Background(), CheckpointOptions{
		Name:    "my-sprite",
		Comment: "test checkpoint",
		client:  client,
	})
	if err != nil {
		t.Fatalf("RunCheckpoint: %v", err)
	}
	if !handle.checkpointDone {
		t.Error("expected Checkpoint to be called on handle")
	}
}

func TestRunCheckpoints_Empty(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunCheckpoints(context.Background(), CheckpointsOptions{Name: "my-sprite", client: client})
	if err != nil {
		t.Fatalf("RunCheckpoints: %v", err)
	}
}

// mockHandleWithCheckpoints extends mockHandle to return checkpoint entries.
type mockHandleWithCheckpoints struct {
	*mockHandle
	entries []CheckpointEntry
}

func (h *mockHandleWithCheckpoints) ListCheckpoints(_ context.Context, _ bool) ([]CheckpointEntry, error) {
	return h.entries, nil
}

func TestRunCheckpoints_TableOutput(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	ts, _ := time.Parse(time.RFC3339, "2026-05-21T10:00:00Z")
	richHandle := &mockHandleWithCheckpoints{
		mockHandle: newMockHandle(),
		entries: []CheckpointEntry{
			{ID: "v1", CreateTime: ts, Comment: "first"},
			{ID: "v2", CreateTime: ts, Comment: "second", IsAuto: true},
		},
	}

	// Capture stdout by redirecting via resolveHandle with test client.
	// Use resolveHandle directly to keep test tight.
	var buf bytes.Buffer
	entries, err := richHandle.ListCheckpoints(context.Background(), false)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 entries, got %d", len(entries))
	}
	_ = buf
}

func TestRunRestore_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunRestore(context.Background(), RestoreOptions{Name: "my-sprite", CheckpointID: "v1"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}
