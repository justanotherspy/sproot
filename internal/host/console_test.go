package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConsole_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunConsole(context.Background(), ConsoleOptions{Name: "my-sprite"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunConsole_CallsHandleConsole(t *testing.T) {
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

	err := RunConsole(context.Background(), ConsoleOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunConsole: %v", err)
	}
}

func TestRunConsole_PropagatesConsoleError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	handle.runErr = context.Canceled
	client := &mockClient{handle: handle}

	err := RunConsole(context.Background(), ConsoleOptions{
		Name:   "my-sprite",
		client: client,
	})
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}
