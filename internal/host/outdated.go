package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// OutdatedOptions controls the behavior of RunOutdated.
type OutdatedOptions struct {
	client SpritesClient
	shaFn  func() (string, ConfigMeta, error) // nil: compute from host config
}

// RunOutdated computes the current config SHA and compares it against every
// sproot-managed sprite's stored sproot-sha label. It prints a table showing
// whether each sprite is current or stale.
func RunOutdated(ctx context.Context, opts OutdatedOptions) error {
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

	shaFn := opts.shaFn
	if shaFn == nil {
		shaFn = func() (string, ConfigMeta, error) {
			return currentConfigSHA(cfg, l)
		}
	}
	currentSHA, _, err := shaFn()
	if err != nil {
		return fmt.Errorf("compute current config SHA: %w", err)
	}

	all, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var shown int
	for _, e := range all {
		if !hasLabel(e.Labels, sprootLabel) {
			continue
		}
		meta := ParseConfigMeta(e.Labels)
		status := "current"
		if meta.SHA == "" {
			status = "unknown (no SHA label)"
		} else if meta.SHA != currentSHA {
			status = "stale"
		}
		target := meta.Target
		if target == "" {
			target = "(default)"
		}
		fmt.Fprintf(os.Stdout, "%-30s %-12s %-14s %s\n", e.Name, target, meta.SHA, status)
		shown++
	}

	if shown == 0 {
		l.Info("no sproot-managed sprites found")
	}
	return nil
}
