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
	NoCheckpoint bool   // skip the pre-push checkpoint
	client       SpritesClient
	shaFn        func() (string, ConfigMeta, error) // nil: compute from host config
}

// RunPush re-runs setup on all sproot-managed sprites (or a specific one).
// It checkpoints before updating by default so the user can restore on failure.
// After each successful push it updates the sprite's metadata labels.
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

	// Compute the current config SHA and base metadata once before pushing.
	shaFn := opts.shaFn
	if shaFn == nil {
		shaFn = func() (string, ConfigMeta, error) {
			return currentConfigSHA(cfg, l)
		}
	}
	currentSHA, baseMeta, err := shaFn()
	if err != nil {
		l.Warnf("could not compute config SHA (labels will not be updated): %v", err)
		currentSHA = ""
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
				err:  pushOne(ctx, client, e.Name, opts, currentSHA, baseMeta, cfg.ConfigPath, l),
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

// pushOne runs setup on a single sprite, optionally checkpointing first,
// then updates the sprite's metadata labels.
func pushOne(ctx context.Context, client SpritesClient, name string, opts PushOptions, sha string, base ConfigMeta, configPath string, l *log.Logger) error {
	handle := client.GetHandle(name)

	if !opts.NoCheckpoint && !opts.DryRun {
		l.Infof("[%s] creating pre-push checkpoint", name)
		if err := handle.Checkpoint(ctx, "pre-push", io.Discard); err != nil {
			l.Warnf("[%s] checkpoint failed (continuing): %v", name, err)
		}
	}

	args := []string{"setup", "--force"}
	// Pass the config source so the in-sprite setup command knows where to fetch config.
	if base.Source == "local" && base.Repo != "" {
		// For local config, upload the host directory to the sprite before running setup,
		// then point setup at the in-sprite copy (the host path is unreachable inside the sprite).
		const spriteLocalConfigDir = "/tmp/sproot-local-config"
		l.Infof("[%s] uploading local config from %s", name, base.Repo)
		if err := uploadDirectory(handle, base.Repo, spriteLocalConfigDir); err != nil {
			return fmt.Errorf("[%s] upload local config: %w", name, err)
		}
		args = append(args, "--local-config", spriteLocalConfigDir)
	} else if base.Repo != "" {
		args = append(args, "--config-repo", base.Repo)
		if base.Ref != "" {
			args = append(args, "--ref", base.Ref)
		}
	}
	if configPath != "" {
		args = append(args, "--config-path", configPath)
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	stdout := &prefixWriter{prefix: "[" + name + "] ", w: os.Stdout}
	stderr := &prefixWriter{prefix: "[" + name + "] ", w: os.Stderr}
	if err := handle.RunCommand("sproot", args, nil, stdout, stderr); err != nil {
		return err
	}

	if !opts.DryRun && sha != "" {
		meta := ConfigMeta{
			Target: opts.Target,
			Source: base.Source,
			Repo:   base.Repo,
			Ref:    base.Ref,
			SHA:    sha,
		}
		if err := handle.SetLabels(ctx, meta.Labels()); err != nil {
			l.Warnf("[%s] set config labels failed: %v", name, err)
		}
	}
	return nil
}

// currentConfigSHA clones/reads the current config and returns its SHA plus
// base metadata (source, repo, ref). Used by RunPush to compute the target SHA.
func currentConfigSHA(cfg *config.HostConfig, l *log.Logger) (string, ConfigMeta, error) {
	if cfg.ConfigSource == "local" {
		dir, err := config.ExpandTilde(cfg.ConfigLocalPath)
		if err != nil {
			return "", ConfigMeta{}, err
		}
		yamlFile := cfg.ConfigPath
		if yamlFile == "" {
			yamlFile = "sproot.yaml"
		}
		raw, err := os.ReadFile(dir + "/" + yamlFile)
		if err != nil {
			return "", ConfigMeta{}, fmt.Errorf("read local config: %w", err)
		}
		meta := ConfigMeta{Source: "local", Repo: dir}
		return ConfigSHA(raw), meta, nil
	}

	// Git source: readEnvBlock clones into its own temp dir and returns the SHA.
	l.Debugf("cloning config repo to compute current SHA")
	_, _, sha, err := readEnvBlock(cfg.ConfigRepo, cfg.ConfigRef, cfg.ConfigPath, l)
	if err != nil {
		return "", ConfigMeta{}, fmt.Errorf("compute current SHA: %w", err)
	}
	meta := ConfigMeta{Source: "git", Repo: cfg.ConfigRepo, Ref: cfg.ConfigRef}
	return sha, meta, nil
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
