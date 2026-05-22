package host

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
	"github.com/justanotherspy/sproot/pkg/table"
	"github.com/justanotherspy/sproot/pkg/tty"
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
	renderCheckpoints(os.Stdout, entries, tty.IsColor(os.Stdout), tty.Width(os.Stdout))
	return nil
}

// renderCheckpoints writes the checkpoint list to w, drawing a colored box on a
// terminal and falling back to tab-separated rows when piped.
func renderCheckpoints(w io.Writer, entries []CheckpointEntry, color bool, width int) {
	if !color {
		for _, e := range entries {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s%s\n",
				e.ID, e.CreateTime.Format("2006-01-02 15:04:05"), e.Comment, autoSuffix(e))
		}
		return
	}

	t := table.Table{
		Columns: []table.Column{
			{Header: "ID", Align: table.AlignLeft},
			{Header: "CREATED", Align: table.AlignLeft},
			{Header: "COMMENT", Align: table.AlignLeft},
		},
		Flex:  2, // COMMENT absorbs slack to fill the terminal width
		Width: width,
		Color: true,
	}
	for _, e := range entries {
		t.Rows = append(t.Rows, []table.Cell{
			{Text: e.ID},
			{Text: e.CreateTime.Format("2006-01-02 15:04:05")},
			{Text: e.Comment + autoSuffix(e)},
		})
	}
	_ = t.Render(w)
}

func autoSuffix(e CheckpointEntry) string {
	if e.IsAuto {
		return " (auto)"
	}
	return ""
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
