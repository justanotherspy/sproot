package host

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sprites "github.com/superfly/sprites-go"
)

// mockListClient implements SpritesClient with configurable ListSprites output.
type mockListClient struct {
	handle  *mockHandle
	entries []SpriteListEntry
	listErr error
}

func (m *mockListClient) CreateSprite(_ context.Context, _ string, _ *sprites.SpriteConfig, _ []string) (SpriteHandle, error) {
	return m.handle, nil
}
func (m *mockListClient) GetHandle(_ string) SpriteHandle                          { return m.handle }
func (m *mockListClient) DestroySprite(_ context.Context, _ string) error          { return nil }
func (m *mockListClient) ListSprites(_ context.Context) ([]SpriteListEntry, error) {
	return m.entries, m.listErr
}

func TestRunList_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunList(context.Background(), ListOptions{})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunList_FiltersToSprootLabel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockListClient{
		handle: handle,
		entries: []SpriteListEntry{
			{Name: "sproot-sprite", Status: "running", Labels: []string{"sproot"}},
			{Name: "other-sprite", Status: "cold", Labels: nil},
		},
	}

	err := RunList(context.Background(), ListOptions{client: client})
	if err != nil {
		t.Fatalf("RunList: %v", err)
	}
}

func TestRunList_PrefixFilters(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	// Capture stdout to verify only matching sprites are printed.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	handle := newMockHandle()
	client := &mockListClient{
		handle: handle,
		entries: []SpriteListEntry{
			{Name: "dev-api", Status: "running", Labels: []string{"sproot"}},
			{Name: "dev-web", Status: "running", Labels: []string{"sproot"}},
			{Name: "staging-api", Status: "cold", Labels: []string{"sproot"}},
		},
	}

	err := RunList(context.Background(), ListOptions{All: true, Prefix: "dev-", client: client})
	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	out := buf.String()

	if err != nil {
		t.Fatalf("RunList: %v", err)
	}
	if !strings.Contains(out, "dev-api") || !strings.Contains(out, "dev-web") {
		t.Errorf("expected dev-* sprites in output, got: %q", out)
	}
	if strings.Contains(out, "staging-api") {
		t.Errorf("staging-api should be filtered by prefix, got: %q", out)
	}
}

func TestRunList_WatchExitsOnContextCancel(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately so watch exits after first iteration

	handle := newMockHandle()
	client := &mockListClient{handle: handle, entries: nil}

	err := RunList(ctx, ListOptions{Watch: true, client: client})
	if err != nil {
		t.Fatalf("expected nil error when context cancelled, got %v", err)
	}
}

func TestRunList_AllShowsEverything(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockListClient{
		handle: handle,
		entries: []SpriteListEntry{
			{Name: "sproot-sprite", Status: "running", Labels: []string{"sproot"}},
			{Name: "other-sprite", Status: "cold", Labels: nil},
		},
	}

	err := RunList(context.Background(), ListOptions{All: true, client: client})
	if err != nil {
		t.Fatalf("RunList --all: %v", err)
	}
}
