package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
)

// ConsoleOptions controls the behavior of RunConsole.
type ConsoleOptions struct {
	Name   string
	client SpritesClient // nil: constructed from token at runtime
}

// RunConsole opens an interactive TTY shell on the named sprite.
func RunConsole(ctx context.Context, opts ConsoleOptions) error {
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
	return handle.Console(nil)
}
