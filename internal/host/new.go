package host

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	sprites "github.com/superfly/sprites-go"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// sprootLabel is attached to every sprite created by sproot new.
// Used by sproot list to filter sprites managed by sproot.
const sprootLabel = "sproot"

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
	NoConsole  bool   // skip opening console after successful setup
	Version    string // build version (from main.version); used to download linux/amd64 binary on non-linux/amd64 hosts

	client           SpritesClient                                                 // nil: use real client (test injection point)
	envBlockReaderFn func(repo, ref, path string, l *log.Logger) ([]string, error) // nil: use readEnvBlock (test injection point)
	binarySrcFn      func(version string) ([]byte, error)                          // nil: auto-detect by arch (test injection point)
}

// RunNew creates a sprite, injects the sproot binary, and runs sproot setup inside it.
func RunNew(ctx context.Context, opts NewOptions) error {
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
	var ghToken string
	if cfg.GHTokenEnv != "" {
		ghToken = os.Getenv(cfg.GHTokenEnv)
		if ghToken == "" {
			l.Warnf("%s (gh_token_env) is not set; GitHub features will be unavailable", cfg.GHTokenEnv)
		}
	}

	// Resolve configPath early so readEnvBlock can find the right yaml file.
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = cfg.ConfigPath
	}

	envReader := opts.envBlockReaderFn
	if envReader == nil {
		envReader = readEnvBlock
	}
	envBlock, err := envReader(cfg.ConfigRepo, cfg.ConfigRef, configPath, l)
	if err != nil {
		return err
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
	handle, err := client.CreateSprite(ctx, opts.Name, spriteCfg, []string{sprootLabel})
	if err != nil {
		return fmt.Errorf("create sprite: %w", err)
	}

	binarySrc := opts.binarySrcFn
	if binarySrc == nil {
		if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
			binarySrc = func(_ string) ([]byte, error) {
				execPath, err := os.Executable()
				if err != nil {
					return nil, fmt.Errorf("find own binary: %w", err)
				}
				return os.ReadFile(execPath)
			}
		} else {
			binarySrc = func(v string) ([]byte, error) {
				l.Infof("host is %s/%s; downloading Linux/amd64 sproot binary for version %s", runtime.GOOS, runtime.GOARCH, v)
				return fetchLinuxAmd64Binary(v)
			}
		}
	}
	binaryData, err := binarySrc(opts.Version)
	if err != nil {
		return fmt.Errorf("get linux/amd64 binary: %w", err)
	}

	l.Info("injecting sproot binary")
	if err := handle.WriteFile("/usr/local/bin/sproot", binaryData, 0755); err != nil {
		return fmt.Errorf("inject sproot binary: %w", err)
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

	// gh_token_env provides a baseline; env block entries are appended after
	// so they take precedence when both define the same variable name.
	var env []string
	if ghToken != "" {
		env = []string{"GH_TOKEN=" + ghToken}
	}
	env = append(env, envBlock...)

	l.Infof("running setup in sprite %s", opts.Name)
	if err := handle.RunCommand("sproot", args, env, os.Stdout, os.Stderr); err != nil {
		return err
	}

	if !opts.NoConsole && !opts.DryRun {
		l.Successf("setup complete; opening console for %s", opts.Name)
		return handle.Console(nil)
	}
	return nil
}

// readEnvBlock clones the config repo at configRepo/configRef into a temp dir,
// loads the sproot.yaml at configPath (or "sproot.yaml" when empty), validates it,
// and resolves each env entry against the host environment. Returns the env slice
// to pass to the sprite. Fails if a required variable is unset on the host.
func readEnvBlock(configRepo, configRef, configPath string, l *log.Logger) ([]string, error) {
	tmpdir, err := os.MkdirTemp("", "sproot-config-*")
	if err != nil {
		return nil, fmt.Errorf("readEnvBlock: create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()

	l.Infof("cloning config repo to resolve env block")
	cloneArgs := []string{"clone", "--depth", "1", "--branch", configRef, configRepo, tmpdir}
	out, err := exec.Command("git", cloneArgs...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("readEnvBlock: git clone: %w\n%s", err, out)
	}

	yamlFile := configPath
	if yamlFile == "" {
		yamlFile = "sproot.yaml"
	}
	sprootCfg, err := config.LoadSprootConfig(filepath.Join(tmpdir, yamlFile))
	if err != nil {
		return nil, fmt.Errorf("readEnvBlock: load sproot.yaml: %w", err)
	}
	if err := config.ValidateSprootConfig(sprootCfg); err != nil {
		return nil, fmt.Errorf("readEnvBlock: validate sproot.yaml: %w", err)
	}

	var env []string
	for _, ev := range sprootCfg.Env {
		val := os.Getenv(ev.From)
		if val == "" {
			if ev.Required {
				return nil, fmt.Errorf("required env var %s (mapped as %s) is not set on host", ev.From, ev.As)
			}
			continue
		}
		env = append(env, ev.As+"="+val)
	}
	return env, nil
}
