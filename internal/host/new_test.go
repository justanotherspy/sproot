package host

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sprites "github.com/superfly/sprites-go"
)

// mockClient implements SpritesClient for testing.
type mockClient struct {
	handle     *mockHandle
	createErr  error
	destroyErr error
}

func (m *mockClient) CreateSprite(_ context.Context, _ string, _ *sprites.SpriteConfig) (SpriteHandle, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	return m.handle, nil
}

func (m *mockClient) GetHandle(_ string) SpriteHandle { return m.handle }

func (m *mockClient) DestroySprite(_ context.Context, _ string) error { return m.destroyErr }

// mockHandle records calls for assertions.
type mockHandle struct {
	writtenFiles map[string][]byte
	readFiles    map[string][]byte
	readErr      error
	lastCmdName  string
	lastCmdArgs  []string
	lastCmdEnv   []string
	runErr       error
}

func newMockHandle() *mockHandle {
	return &mockHandle{
		writtenFiles: make(map[string][]byte),
		readFiles:    make(map[string][]byte),
	}
}

func (h *mockHandle) WriteFile(path string, data []byte, _ fs.FileMode) error {
	h.writtenFiles[path] = data
	return nil
}

func (h *mockHandle) ReadFile(path string) ([]byte, error) {
	if h.readErr != nil {
		return nil, h.readErr
	}
	data, ok := h.readFiles[path]
	if !ok {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func (h *mockHandle) RunCommand(name string, args, env []string, _, _ io.Writer) error {
	h.lastCmdName = name
	h.lastCmdArgs = args
	h.lastCmdEnv = env
	return h.runErr
}

func writeHostConfig(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "config")
	if err := os.MkdirAll(dir, 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunNew_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "")
	t.Setenv("MY_GH_TOKEN", "gh-tok")

	err := RunNew(context.Background(), NewOptions{Name: "test"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunNew_NoGHToken_Succeeds(t *testing.T) {
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

	err := RunNew(context.Background(), NewOptions{Name: "test", client: client})
	if err != nil {
		t.Fatalf("expected success without gh_token_env, got: %v", err)
	}
	for _, e := range handle.lastCmdEnv {
		if strings.HasPrefix(e, "GH_TOKEN=") {
			t.Errorf("GH_TOKEN should not be forwarded when not configured, got: %v", handle.lastCmdEnv)
		}
	}
}

func TestRunNew_GHTokenEnvSetButEmpty_Succeeds(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunNew(context.Background(), NewOptions{Name: "test", client: client})
	if err != nil {
		t.Fatalf("expected success when gh token env var is empty, got: %v", err)
	}
	for _, e := range handle.lastCmdEnv {
		if strings.HasPrefix(e, "GH_TOKEN=") {
			t.Errorf("GH_TOKEN should not be forwarded when empty, got: %v", handle.lastCmdEnv)
		}
	}
}

func TestRunNew_InjectsBinaryAndForwardsGHToken(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "gh-secret")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunNew(context.Background(), NewOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}

	if _, ok := handle.writtenFiles["/usr/local/bin/sproot"]; !ok {
		t.Error("expected /usr/local/bin/sproot to be written")
	}

	if handle.lastCmdName != "sproot" {
		t.Errorf("command name: got %q, want sproot", handle.lastCmdName)
	}
	if len(handle.lastCmdArgs) < 2 || handle.lastCmdArgs[0] != "setup" {
		t.Errorf("command args: %v", handle.lastCmdArgs)
	}

	found := false
	for _, e := range handle.lastCmdEnv {
		if e == "GH_TOKEN=gh-secret" {
			found = true
		}
	}
	if !found {
		t.Errorf("GH_TOKEN not forwarded in env: %v", handle.lastCmdEnv)
	}
}

func TestRunNew_ForwardsConfigPath(t *testing.T) {
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

	err := RunNew(context.Background(), NewOptions{
		Name:       "my-sprite",
		ConfigPath: "configs/dev.yaml",
		client:     client,
	})
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if !containsSeq(handle.lastCmdArgs, "--config-path", "configs/dev.yaml") {
		t.Errorf("--config-path not forwarded: %v", handle.lastCmdArgs)
	}
}

func TestRunNew_ConfigPathFromHostConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
config_path: infra/sproot.yaml
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunNew(context.Background(), NewOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if !containsSeq(handle.lastCmdArgs, "--config-path", "infra/sproot.yaml") {
		t.Errorf("--config-path from host config not forwarded: %v", handle.lastCmdArgs)
	}
}

func TestRunNew_CLIConfigPathOverridesHostConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
config_path: infra/sproot.yaml
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunNew(context.Background(), NewOptions{
		Name:       "my-sprite",
		ConfigPath: "override.yaml",
		client:     client,
	})
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}
	if !containsSeq(handle.lastCmdArgs, "--config-path", "override.yaml") {
		t.Errorf("CLI --config-path should override host config: %v", handle.lastCmdArgs)
	}
}

func TestRunNew_ForwardsSetupFlags(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "gh-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunNew(context.Background(), NewOptions{
		Name:   "my-sprite",
		Only:   "apt",
		Force:  true,
		DryRun: true,
		client: client,
	})
	if err != nil {
		t.Fatalf("RunNew: %v", err)
	}

	args := handle.lastCmdArgs
	if !containsSeq(args, "--only", "apt") {
		t.Errorf("--only not forwarded: %v", args)
	}
	if !contains(args, "--force") {
		t.Errorf("--force not forwarded: %v", args)
	}
	if !contains(args, "--dry-run") {
		t.Errorf("--dry-run not forwarded: %v", args)
	}
}

func containsSeq(slice []string, a, b string) bool {
	for i := 0; i+1 < len(slice); i++ {
		if slice[i] == a && slice[i+1] == b {
			return true
		}
	}
	return false
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
