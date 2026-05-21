package host

import (
	"context"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// CheckpointOptions controls the behavior of RunCheckpoint.
type CheckpointOptions struct {
	Name    string
	Comment string
	client  SpritesClient // nil: constructed from token at runtime
}

// CheckpointsOptions controls the behavior of RunCheckpoints.
type CheckpointsOptions struct {
	Name        string
	IncludeAuto bool
	client      SpritesClient // nil: constructed from token at runtime
}

// RestoreOptions controls the behavior of RunRestore.
type RestoreOptions struct {
	Name         string
	CheckpointID string
	client       SpritesClient // nil: constructed from token at runtime
}

// RunCheckpoint creates a checkpoint for the named sprite, streaming progress to stdout.
func RunCheckpoint(ctx context.Context, opts CheckpointOptions) error {
	l := log.Stderr()
	handle, err := resolveHandle(opts.client, opts.Name)
	if err != nil {
		return err
	}
	l.Infof("creating checkpoint for %s", opts.Name)
	if err := handle.Checkpoint(ctx, opts.Comment, os.Stdout); err != nil {
		return fmt.Errorf("checkpoint: %w", err)
	}
	l.Success("checkpoint created")
	return nil
}

// RunCheckpoints lists checkpoints for the named sprite.
func RunCheckpoints(ctx context.Context, opts CheckpointsOptions) error {
	handle, err := resolveHandle(opts.client, opts.Name)
	if err != nil {
		return err
	}
	entries, err := handle.ListCheckpoints(ctx, opts.IncludeAuto)
	if err != nil {
		return fmt.Errorf("list checkpoints: %w", err)
	}
	if len(entries) == 0 {
		_, err = fmt.Fprintln(os.Stdout, "no checkpoints found")
		return err
	}
	tw := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(tw, "ID\tCREATED\tCOMMENT")
	for _, e := range entries {
		auto := ""
		if e.IsAuto {
			auto = " (auto)"
		}
		_, _ = fmt.Fprintf(tw, "%s\t%s\t%s%s\n",
			e.ID,
			e.CreateTime.Format("2006-01-02 15:04:05"),
			e.Comment,
			auto,
		)
	}
	return tw.Flush()
}

// RunRestore restores the named sprite from a checkpoint, streaming progress to stdout.
func RunRestore(ctx context.Context, opts RestoreOptions) error {
	l := log.Stderr()
	handle, err := resolveHandle(opts.client, opts.Name)
	if err != nil {
		return err
	}
	l.Infof("restoring %s from checkpoint %s", opts.Name, opts.CheckpointID)
	if err := handle.Restore(ctx, opts.CheckpointID, os.Stdout); err != nil {
		return fmt.Errorf("restore: %w", err)
	}
	l.Successf("restored %s", opts.Name)
	return nil
}

// resolveHandle loads host config, validates the token, and returns a SpriteHandle.
// client may be nil (production path) or a test double.
func resolveHandle(client SpritesClient, name string) (SpriteHandle, error) {
	if client != nil {
		return client.GetHandle(name), nil
	}
	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return nil, err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return nil, err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return nil, err
	}
	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return nil, fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}
	return NewClient(token).GetHandle(name), nil
}
