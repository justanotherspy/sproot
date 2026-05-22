package host

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sprites "github.com/superfly/sprites-go"

	"github.com/justanotherspy/sproot/internal/phase/modules"
)

func TestRunDestroy_ReadsKeyFileAndCallsGitHubAPI(t *testing.T) {
	var deletedPaths []string
	ghServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deletedPaths = append(deletedPaths, r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ghServer.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "gh-tok")

	ids := modules.GHKeyIDs{AuthKeyID: 11, SigningKeyID: 22}
	idsJSON, _ := json.Marshal(ids)

	handle := newMockHandle()
	handle.readFiles[ghKeyIDsSpritePath] = idsJSON

	destroyCalled := false
	client := &mockClientTracked{
		handle:          handle,
		onDestroySprite: func() { destroyCalled = true },
	}

	err := RunDestroy(context.Background(), DestroyOptions{
		Name:      "my-sprite",
		client:    client,
		ghAPIBase: ghServer.URL,
	})
	if err != nil {
		t.Fatalf("RunDestroy: %v", err)
	}
	if !destroyCalled {
		t.Error("expected DestroySprite to be called")
	}
	if len(deletedPaths) != 2 {
		t.Errorf("expected 2 GitHub key deletions, got %d: %v", len(deletedPaths), deletedPaths)
	}
}

func TestRunDestroy_AbsentKeyFileStillDestroys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "gh-tok")

	handle := newMockHandle()
	// readErr causes ReadFile to fail, simulating absent key file.
	handle.readErr = os.ErrNotExist

	destroyCalled := false
	client := &mockClientTracked{
		handle:          handle,
		onDestroySprite: func() { destroyCalled = true },
	}

	err := RunDestroy(context.Background(), DestroyOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunDestroy: %v", err)
	}
	if !destroyCalled {
		t.Error("expected DestroySprite to be called even when key file is absent")
	}
}

func TestRunDestroy_ForceSkipsConfirmation(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
	t.Setenv("MY_GH_TOKEN", "gh-tok")

	handle := newMockHandle()
	handle.readErr = os.ErrNotExist
	destroyCalled := false
	client := &mockClientTracked{
		handle:          handle,
		onDestroySprite: func() { destroyCalled = true },
	}

	err := RunDestroy(context.Background(), DestroyOptions{
		Name:   "my-sprite",
		Force:  true,
		client: client,
	})
	if err != nil {
		t.Fatalf("RunDestroy --force: %v", err)
	}
	if !destroyCalled {
		t.Error("expected DestroySprite to be called with --force")
	}
}

func TestRunDestroy_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
gh_token_env: MY_GH_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunDestroy(context.Background(), DestroyOptions{Name: "my-sprite"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

// mockClientTracked lets tests observe when DestroySprite is called.
type mockClientTracked struct {
	handle          *mockHandle
	onDestroySprite func()
}

func (m *mockClientTracked) CreateSprite(_ context.Context, _ string, _ *sprites.SpriteConfig, _ []string) (SpriteHandle, error) {
	return m.handle, nil
}
func (m *mockClientTracked) GetHandle(_ string) SpriteHandle { return m.handle }
func (m *mockClientTracked) DestroySprite(_ context.Context, _ string) error {
	if m.onDestroySprite != nil {
		m.onDestroySprite()
	}
	return nil
}
func (m *mockClientTracked) ListSprites(_ context.Context) ([]SpriteListEntry, error) {
	return nil, nil
}
