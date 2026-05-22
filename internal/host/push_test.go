package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	sprites "github.com/superfly/sprites-go"
)

// pushMockClient extends mockClient with a configurable sprite list.
type pushMockClient struct {
	sprites    []SpriteListEntry
	handle     *mockHandle
	listErr    error
	createErr  error
	destroyErr error
}

func (m *pushMockClient) CreateSprite(_ context.Context, _ string, _ *sprites.SpriteConfig, _ []string) (SpriteHandle, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.handle, nil
}
func (m *pushMockClient) GetHandle(_ string) SpriteHandle { return m.handle }
func (m *pushMockClient) DestroySprite(_ context.Context, _ string) error { return m.destroyErr }
func (m *pushMockClient) ListSprites(_ context.Context) ([]SpriteListEntry, error) {
	return m.sprites, m.listErr
}

func noopSHAFn() (string, ConfigMeta, error) {
	return "abc123def456", ConfigMeta{Source: "git", Repo: "git@github.com:u/r.git", Ref: "main"}, nil
}

func TestRunPush_AllSprites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
			{Name: "sprite-b", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		NoCheckpoint: true,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Verify setup --force was called.
	if handle.lastCmdName != "sproot" {
		t.Errorf("cmd name: got %q, want sproot", handle.lastCmdName)
	}
	if len(handle.lastCmdArgs) == 0 || handle.lastCmdArgs[0] != "setup" {
		t.Errorf("cmd args: got %v, want [setup ...]", handle.lastCmdArgs)
	}
	found := false
	for _, a := range handle.lastCmdArgs {
		if a == "--force" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --force in args %v", handle.lastCmdArgs)
	}
}

func TestRunPush_WithCheckpoint(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		NoCheckpoint: false,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handle.checkpointDone {
		t.Error("expected checkpoint to be called before push")
	}
}

func TestRunPush_SpecificSprite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
			{Name: "sprite-b", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		SpriteName:   "sprite-a",
		NoCheckpoint: true,
		client:       client,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunPush_SpriteNotFound(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		SpriteName:   "nonexistent",
		NoCheckpoint: true,
		client:       client,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("error %q missing expected message", err.Error())
	}
}

func TestRunPush_WithTarget(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		Target:       "web",
		NoCheckpoint: true,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for i, a := range handle.lastCmdArgs {
		if a == "--target" && i+1 < len(handle.lastCmdArgs) && handle.lastCmdArgs[i+1] == "web" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --target web in args %v", handle.lastCmdArgs)
	}
}

func TestRunPush_NoSprites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "other", Labels: []string{"other-label"}},
		},
		handle: newMockHandle(),
	}

	err := RunPush(context.Background(), PushOptions{
		NoCheckpoint: true,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Errorf("expected no error when no sprites found, got %v", err)
	}
}

func TestRunPush_DryRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		DryRun:       true,
		NoCheckpoint: true,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	found := false
	for _, a := range handle.lastCmdArgs {
		if a == "--dry-run" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected --dry-run in args %v", handle.lastCmdArgs)
	}
	// Labels must NOT be updated during a dry run.
	if _, ok := handle.writtenFiles["__labels__"]; ok {
		t.Error("SetLabels must not be called during a dry run")
	}
}

func TestRunPush_LabelsUpdated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "sprite-a", Labels: []string{sprootLabel}},
		},
		handle: handle,
	}

	err := RunPush(context.Background(), PushOptions{
		NoCheckpoint: true,
		client:       client,
		shaFn:        noopSHAFn,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	raw, ok := handle.writtenFiles["__labels__"]
	if !ok {
		t.Fatal("expected SetLabels to be called after a successful push")
	}
	labelStr := string(raw)
	if !strings.Contains(labelStr, "sproot-sha=abc123def456") {
		t.Errorf("labels missing expected SHA; got %s", labelStr)
	}
	if !strings.Contains(labelStr, "sproot-source=git") {
		t.Errorf("labels missing expected source; got %s", labelStr)
	}
}

func TestPrefixWriter(t *testing.T) {
	var buf strings.Builder
	pw := &prefixWriter{prefix: "[test] ", w: &buf}

	_, _ = pw.Write([]byte("hello\nworld\n"))
	got := buf.String()
	want := "[test] hello\n[test] world\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrefixWriter_PartialLine(t *testing.T) {
	var buf strings.Builder
	pw := &prefixWriter{prefix: "[x] ", w: &buf}

	_, _ = pw.Write([]byte("hel"))
	_, _ = pw.Write([]byte("lo\n"))
	got := buf.String()
	want := "[x] hello\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
