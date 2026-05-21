package host

import (
	"context"
	"path/filepath"
	"testing"
)

func TestRunOutdated_ShowsCurrentAndStale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
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

func TestRunOutdated_NoSprites(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
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
