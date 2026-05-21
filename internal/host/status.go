package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/internal/sprite"
)

// spriteStatePath is the path to the sproot state file inside a sprite.
// Sprites run as root so XDG config resolves to /root/.config.
const spriteStatePath = "/root/.config/sproot/state.json"

// StatusOptions controls the behavior of RunStatus.
type StatusOptions struct {
	Name   string
	Host   bool          // read state from sprite filesystem instead of exec'ing in-sprite
	client SpritesClient // nil: constructed from token at runtime
}

// RunStatus prints phase status for the named sprite.
// Without --host it runs "sproot setup --status" inside the sprite (requires exec).
// With --host it reads the state file from the sprite filesystem directly.
func RunStatus(ctx context.Context, opts StatusOptions) error {
	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return err
	}

	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	handle := client.GetHandle(opts.Name)

	if opts.Host {
		return runStatusHost(handle)
	}
	return handle.RunCommand("sproot", []string{"setup", "--status"}, nil, os.Stdout, os.Stderr)
}

// runStatusHost reads the state.json from the sprite filesystem and renders it locally.
func runStatusHost(handle SpriteHandle) error {
	data, err := handle.ReadFile(spriteStatePath)
	if err != nil {
		return fmt.Errorf("read state from sprite: %w (has sproot setup been run?)", err)
	}
	var state phase.State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("parse state file: %w", err)
	}
	return sprite.RenderStatus(&state, "", os.Stdout)
}
