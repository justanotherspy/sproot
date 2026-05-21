package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// ListOptions controls the behavior of RunList.
type ListOptions struct {
	All    bool          // show all sprites, not just sproot-managed ones
	client SpritesClient // nil: constructed from token at runtime
}

// RunList lists sprites. Without --all, only sprites tagged with the sproot
// label (created by sproot new) are shown.
func RunList(ctx context.Context, opts ListOptions) error {
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

	entries, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var shown int
	for _, e := range entries {
		if !opts.All && !hasLabel(e.Labels, sprootLabel) {
			continue
		}
		_, _ = fmt.Fprintf(os.Stdout, "%-30s %s\n", e.Name, e.Status)
		l.Debugf("labels: %v", e.Labels)
		shown++
	}

	if shown == 0 {
		if opts.All {
			l.Info("no sprites found")
		} else {
			l.Info("no sproot-managed sprites found (use --all to see all sprites)")
		}
	}
	return nil
}

// hasLabel reports whether labels contains the target string.
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}
