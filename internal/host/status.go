package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
)

// StatusOptions controls the behavior of RunStatus.
type StatusOptions struct {
	Name   string
	client SpritesClient // nil: constructed from token at runtime
}

// RunStatus runs "sproot setup --status" inside the named sprite and streams its output.
func RunStatus(ctx context.Context, opts StatusOptions) error {
	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := config.LoadHostConfig(cfgPath)
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
	return handle.RunCommand("sproot", []string{"setup", "--status"}, nil, os.Stdout, os.Stderr)
}
