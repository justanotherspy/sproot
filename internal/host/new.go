package host

import (
	"context"
	"fmt"
	"os"

	sprites "github.com/superfly/sprites-go"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// NewOptions controls the behavior of RunNew.
type NewOptions struct {
	Name       string
	RamMB      int
	CPUs       int
	Region     string
	ConfigPath string // path to config file within the repo; overrides host config; empty uses host config or "sproot.yaml"
	Only       string
	Force      bool
	DryRun     bool

	client SpritesClient // nil: constructed from token at runtime (test injection point)
}

// RunNew creates a sprite, injects the sproot binary, and runs sproot setup inside it.
func RunNew(ctx context.Context, opts NewOptions) error {
	l := log.Stderr()

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
	var ghToken string
	if cfg.GHTokenEnv != "" {
		ghToken = os.Getenv(cfg.GHTokenEnv)
		if ghToken == "" {
			l.Warnf("%s (gh_token_env) is not set; GitHub features will be unavailable", cfg.GHTokenEnv)
		}
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	spriteCfg := &sprites.SpriteConfig{}
	if opts.RamMB > 0 {
		spriteCfg.RamMB = opts.RamMB
	}
	if opts.CPUs > 0 {
		spriteCfg.CPUs = opts.CPUs
	}
	if opts.Region != "" {
		spriteCfg.Region = opts.Region
	}

	l.Infof("creating sprite %s", opts.Name)
	handle, err := client.CreateSprite(ctx, opts.Name, spriteCfg)
	if err != nil {
		return fmt.Errorf("create sprite: %w", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("find own binary: %w", err)
	}
	binaryData, err := os.ReadFile(execPath)
	if err != nil {
		return fmt.Errorf("read own binary: %w", err)
	}

	l.Info("injecting sproot binary")
	if err := handle.WriteFile("/usr/local/bin/sproot", binaryData, 0755); err != nil {
		return fmt.Errorf("inject sproot binary: %w", err)
	}

	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = cfg.ConfigPath
	}

	args := []string{"setup", "--config-repo", cfg.ConfigRepo, "--ref", cfg.ConfigRef}
	if configPath != "" {
		args = append(args, "--config-path", configPath)
	}
	if opts.Only != "" {
		args = append(args, "--only", opts.Only)
	}
	if opts.Force {
		args = append(args, "--force")
	}
	if opts.DryRun {
		args = append(args, "--dry-run")
	}

	var env []string
	if ghToken != "" {
		env = []string{"GH_TOKEN=" + ghToken}
	}

	l.Infof("running setup in sprite %s", opts.Name)
	return handle.RunCommand("sproot", args, env, os.Stdout, os.Stderr)
}
