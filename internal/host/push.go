package host

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// PushOptions controls the behavior of RunPush.
type PushOptions struct {
	SpriteName   string // empty = all sproot-managed sprites
	Target       string // passed as --target to sproot setup
	DryRun       bool
	NoCheckpoint bool // skip the pre-push checkpoint
	client       SpritesClient
}

// RunPush re-runs setup on all sproot-managed sprites (or a specific one).
// It checkpoints before updating by default so the user can restore on failure.
func RunPush(ctx context.Context, opts PushOptions) error {
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

	all, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var targets []SpriteListEntry
	for _, e := range all {
		if !hasLabel(e.Labels, sprootLabel) {
			continue
		}
		if opts.SpriteName != "" && e.Name != opts.SpriteName {
			continue
		}
		targets = append(targets, e)
	}

	if len(targets) == 0 {
		if opts.SpriteName != "" {
			return fmt.Errorf("sprite %q not found or not managed by sproot", opts.SpriteName)
		}
		l.Info("no sproot-managed sprites found")
		return nil
	}

	l.Infof("pushing to %d sprite(s)", len(targets))

	type result struct {
		name string
		err  error
	}
	results := make([]result, len(targets))

	var wg sync.WaitGroup
	for i, entry := range targets {
		wg.Add(1)
		go func(idx int, e SpriteListEntry) {
			defer wg.Done()
			results[idx] = result{
				name: e.Name,
				err:  pushOne(ctx, client, e.Name, opts, l),
			}
		}(i, entry)
	}
	wg.Wait()

	var failed int
	for _, r := range results {
		if r.err != nil {
			l.Errorf("push to %s failed: %v", r.name, r.err)
			failed++
		} else {
			l.Successf("push to %s succeeded", r.name)
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d sprite(s) failed", failed, len(targets))
	}
	return nil
}

// pushOne runs setup on a single sprite, optionally checkpointing first.
func pushOne(ctx context.Context, client SpritesClient, name string, opts PushOptions, l *log.Logger) error {
	handle := client.GetHandle(name)

	if !opts.NoCheckpoint && !opts.DryRun {
		l.Infof("[%s] creating pre-push checkpoint", name)
		if err := handle.Checkpoint(ctx, "pre-push", io.Discard); err != nil {
			l.Warnf("[%s] checkpoint failed (continuing): %v", name, err)
		}
	}

	args := []string{"setup", "--force"}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	stdout := &prefixWriter{prefix: "[" + name + "] ", w: os.Stdout}
	stderr := &prefixWriter{prefix: "[" + name + "] ", w: os.Stderr}
	return handle.RunCommand("sproot", args, nil, stdout, stderr)
}

// prefixWriter wraps an io.Writer and prepends a prefix to each line of output.
type prefixWriter struct {
	prefix  string
	w       io.Writer
	partial []byte
}

func (pw *prefixWriter) Write(p []byte) (int, error) {
	total := len(p)
	buf := append(pw.partial, p...)
	pw.partial = nil
	for {
		idx := bytes.IndexByte(buf, '\n')
		if idx < 0 {
			pw.partial = append(pw.partial, buf...)
			return total, nil
		}
		line := buf[:idx+1]
		if _, err := fmt.Fprintf(pw.w, "%s%s", pw.prefix, line); err != nil {
			return 0, err
		}
		buf = buf[idx+1:]
	}
}
