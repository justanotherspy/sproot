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

// RunUpgrade upgrades the named sprite to the latest version.
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
	l.Infof("upgrading sprite %s", opts.Name)
	if err := handle.Upgrade(ctx); err != nil {
		return fmt.Errorf("upgrade sprite: %w", err)
	}
	l.Successf("upgraded %s", opts.Name)
	return nil
}
