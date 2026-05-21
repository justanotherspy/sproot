package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUpgrade_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunUpgrade(context.Background(), UpgradeOptions{Name: "my-sprite"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunUpgrade_CallsUpgrade(t *testing.T) {
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

	err := RunUpgrade(context.Background(), UpgradeOptions{Name: "my-sprite", client: client})
	if err != nil {
		t.Fatalf("RunUpgrade: %v", err)
	}
}
