package host

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunOutdated_ShowsCurrentAndStale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	currentSHA := "abc123def456"
	staleSHA := "000000000000"

	client := &pushMockClient{
		sprites: []SpriteListEntry{
			{Name: "up-to-date", Labels: []string{"sproot", "sproot-sha=" + currentSHA, "sproot-target="}},
			{Name: "stale-sprite", Labels: []string{"sproot", "sproot-sha=" + staleSHA, "sproot-target=web"}},
			{Name: "no-sha", Labels: []string{"sproot"}},
		},
		handle: newMockHandle(),
	}

	err := RunOutdated(context.Background(), OutdatedOptions{
		client: client,
		shaFn: func() (string, ConfigMeta, error) {
			return currentSHA, ConfigMeta{Source: "git"}, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunOutdated_SHAFnError(t *testing.T) {
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
			{Name: "sprite-a", Labels: []string{"sproot", "sproot-sha=abc"}},
		},
		handle: newMockHandle(),
	}

	err := RunOutdated(context.Background(), OutdatedOptions{
		client: client,
		shaFn: func() (string, ConfigMeta, error) {
			return "", ConfigMeta{}, fmt.Errorf("config clone failed")
		},
	})
	if err == nil {
		t.Fatal("expected error when shaFn fails, got nil")
	}
	if !strings.Contains(err.Error(), "config clone failed") {
		t.Errorf("error %q does not mention root cause", err.Error())
	}
}

func TestRunOutdated_ListError(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	client := &pushMockClient{
		listErr: fmt.Errorf("api unavailable"),
		handle:  newMockHandle(),
	}

	err := RunOutdated(context.Background(), OutdatedOptions{
		client: client,
		shaFn: func() (string, ConfigMeta, error) {
			return "abc123", ConfigMeta{}, nil
		},
	})
	if err == nil {
		t.Fatal("expected error when ListSprites fails, got nil")
	}
	if !strings.Contains(err.Error(), "api unavailable") {
		t.Errorf("error %q does not mention root cause", err.Error())
	}
}

func TestRunOutdated_NoSprites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	client := &pushMockClient{
		sprites: []SpriteListEntry{},
		handle:  newMockHandle(),
	}

	err := RunOutdated(context.Background(), OutdatedOptions{
		client: client,
		shaFn: func() (string, ConfigMeta, error) {
			return "abc123", ConfigMeta{}, nil
		},
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
