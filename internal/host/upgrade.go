package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// UpgradeOptions controls the behavior of RunUpgrade.
type UpgradeOptions struct {
	Name   string
	client SpritesClient // nil: constructed from token at runtime
}

// RunUpgrade runs "sprite upgrade" inside the named sprite, upgrading the sprite CLI binary.
func RunUpgrade(ctx context.Context, opts UpgradeOptions) error {
	l := log.Stderr()

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
	l.Infof("upgrading sprite CLI in %s", opts.Name)
	return handle.RunCommand("sprite", []string{"upgrade"}, nil, os.Stdout, os.Stderr)
}
