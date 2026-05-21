package host

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// ListOptions controls the behavior of RunList.
type ListOptions struct {
	All    bool          // show all sprites, not just sproot-managed ones
	Prefix string        // filter sprites by name prefix (empty means no filter)
	Watch  bool          // poll and refresh every 2 seconds
	client SpritesClient // nil: constructed from token at runtime
}

// RunList lists sprites. Without --all, only sprites tagged with the sproot
// label (created by sproot new) are shown. --prefix filters by name prefix.
// --watch polls and refreshes every 2 seconds until the context is cancelled.
func RunList(ctx context.Context, opts ListOptions) error {
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

	if opts.Watch {
		for {
			fmt.Print("\033[H\033[2J")
			if err := printList(ctx, client, opts); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
		}
	}
	return printList(ctx, client, opts)
}

func printList(ctx context.Context, client SpritesClient, opts ListOptions) error {
	l := log.Stderr()

	entries, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var shown int
	for _, e := range entries {
		if !opts.All && !hasLabel(e.Labels, sprootLabel) {
			continue
		}
		if opts.Prefix != "" && !strings.HasPrefix(e.Name, opts.Prefix) {
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
