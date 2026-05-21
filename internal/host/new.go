package host

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"golang.org/x/term"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// sprootLabel is attached to every sprite created by sproot new.
// Used by sproot list to filter sprites managed by sproot.
const sprootLabel = "sproot"

// NewOptions controls the behavior of RunNew.
type NewOptions struct {
	Name        string
	ConfigPath  string // path to config file within the repo; overrides host config; empty uses host config or "sproot.yaml"
	Target      string // named target from sproot.yaml; empty uses default/flat phases
	LocalConfig string // host directory path to use instead of git clone (overrides config_source in host config)
	Only        string
	Force       bool
	DryRun      bool
	SkipConsole bool   // skip opening console after successful setup
	Version     string // build version (from main.version); used to download linux/amd64 binary on non-linux/amd64 hosts

	client              SpritesClient                                                                     // nil: use real client (test injection point)
	envBlockReaderFn    func(repo, ref, path string, l *log.Logger) ([]string, *config.SprootConfig, error) // nil: use readEnvBlock (test injection point)
	localCfgReaderFn    func(dir, configPath string, l *log.Logger) ([]string, *config.SprootConfig, error) // nil: use readLocalEnvBlock (test injection point)
	binarySrcFn         func(version string) ([]byte, error)                                               // nil: auto-detect by arch (test injection point)
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

	// Resolve configPath early so readers can find the right yaml file.
	configPath := opts.ConfigPath
	if configPath == "" {
		configPath = cfg.ConfigPath
	}

	// Determine the effective local config directory (flag takes precedence over host config).
	localConfigDir := opts.LocalConfig
	if localConfigDir == "" && cfg.ConfigSource == "local" {
		localConfigDir = cfg.ConfigLocalPath
	}

	var (
		envBlock  []string
		sprootCfg *config.SprootConfig
	)
	if localConfigDir != "" {
		expanded, err := config.ExpandTilde(localConfigDir)
		if err != nil {
			return fmt.Errorf("expand local config path: %w", err)
		}
		localConfigDir = expanded
		localReader := opts.localCfgReaderFn
		if localReader == nil {
			localReader = readLocalEnvBlock
		}
		envBlock, sprootCfg, err = localReader(localConfigDir, configPath, l)
		if err != nil {
			return err
		}
	} else {
		envReader := opts.envBlockReaderFn
		if envReader == nil {
			envReader = readEnvBlock
		}
		envBlock, sprootCfg, err = envReader(cfg.ConfigRepo, cfg.ConfigRef, configPath, l)
		if err != nil {
			return err
		}
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	l.Infof("creating sprite %s", opts.Name)
	handle, err := client.CreateSprite(ctx, opts.Name, nil, []string{sprootLabel})
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

	var args []string
	if localConfigDir != "" {
		const spriteLocalConfigDir = "/tmp/sproot-local-config"
		l.Infof("uploading local config from %s", localConfigDir)
		if err := uploadDirectory(handle, localConfigDir, spriteLocalConfigDir); err != nil {
			return fmt.Errorf("upload local config: %w", err)
		}
		args = []string{"setup", "--local-config", spriteLocalConfigDir}
		if configPath != "" {
			args = append(args, "--config-path", configPath)
		}
	} else {
		args = []string{"setup", "--config-repo", cfg.ConfigRepo, "--ref", cfg.ConfigRef}
		if configPath != "" {
			args = append(args, "--config-path", configPath)
		}
	}
	if opts.Target != "" {
		args = append(args, "--target", opts.Target)
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

	if !opts.DryRun && sprootCfg != nil && sprootCfg.CheckpointAfterSetup {
		l.Info("creating post-setup checkpoint")
		if err := handle.Checkpoint(ctx, "sproot setup", os.Stdout); err != nil {
			l.Warnf("checkpoint failed: %v", err)
		} else {
			l.Success("checkpoint created")
		}
	}

	if !opts.SkipConsole && !opts.DryRun && term.IsTerminal(int(os.Stdin.Fd())) {
		l.Successf("setup complete; opening console for %s", opts.Name)
		return handle.Console(nil)
	}
	return nil
}

// readLocalEnvBlock loads sproot.yaml from a local directory (no git clone),
// validates it, and resolves the env block. Used when config_source=local.
func readLocalEnvBlock(dir, configPath string, l *log.Logger) ([]string, *config.SprootConfig, error) {
	l.Debugf("reading local config from %s", dir)
	yamlFile := configPath
	if yamlFile == "" {
		yamlFile = "sproot.yaml"
	}
	sprootCfg, err := config.LoadSprootConfig(filepath.Join(dir, yamlFile))
	if err != nil {
		return nil, nil, fmt.Errorf("readLocalEnvBlock: load sproot.yaml: %w", err)
	}
	if err := config.ValidateSprootConfig(sprootCfg); err != nil {
		return nil, nil, fmt.Errorf("readLocalEnvBlock: validate sproot.yaml: %w", err)
	}
	var env []string
	for _, ev := range sprootCfg.Env {
		val := os.Getenv(ev.From)
		if val == "" {
			if ev.Required {
				return nil, nil, fmt.Errorf("required env var %s (mapped as %s) is not set on host", ev.From, ev.As)
			}
			continue
		}
		env = append(env, ev.As+"="+val)
	}
	return env, sprootCfg, nil
}

// uploadDirectory walks srcDir and writes each file to destDir in the sprite,
// skipping .git and any entry whose name starts with ".".
func uploadDirectory(handle SpriteHandle, srcDir, destDir string) error {
	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		name := info.Name()
		if name == ".git" || (name[0] == '.' && info.IsDir()) {
			return filepath.SkipDir
		}
		if name[0] == '.' {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read %s: %w", path, err)
		}
		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		dest := destDir + "/" + filepath.ToSlash(rel)
		perm := info.Mode().Perm()
		if err := handle.WriteFile(dest, data, perm); err != nil {
			return fmt.Errorf("upload %s: %w", rel, err)
		}
		return nil
	})
}

// readEnvBlock clones the config repo at configRepo/configRef into a temp dir,
// loads the sproot.yaml at configPath (or "sproot.yaml" when empty), validates it,
// and resolves each env entry against the host environment. Returns the env slice
// to pass to the sprite and the parsed SprootConfig. Fails if a required variable
// is unset on the host.
func readEnvBlock(configRepo, configRef, configPath string, l *log.Logger) ([]string, *config.SprootConfig, error) {
	tmpdir, err := os.MkdirTemp("", "sproot-config-*")
	if err != nil {
		return nil, nil, fmt.Errorf("readEnvBlock: create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()

	l.Infof("cloning config repo to resolve env block")
	cloneArgs := []string{"clone", "--depth", "1", "--branch", configRef, configRepo, tmpdir}
	out, err := exec.Command("git", cloneArgs...).CombinedOutput()
	if err != nil {
		return nil, nil, fmt.Errorf("readEnvBlock: git clone: %w\n%s", err, out)
	}

	yamlFile := configPath
	if yamlFile == "" {
		yamlFile = "sproot.yaml"
	}
	sprootCfg, err := config.LoadSprootConfig(filepath.Join(tmpdir, yamlFile))
	if err != nil {
		return nil, nil, fmt.Errorf("readEnvBlock: load sproot.yaml: %w", err)
	}
	if err := config.ValidateSprootConfig(sprootCfg); err != nil {
		return nil, nil, fmt.Errorf("readEnvBlock: validate sproot.yaml: %w", err)
	}

	var env []string
	for _, ev := range sprootCfg.Env {
		val := os.Getenv(ev.From)
		if val == "" {
			if ev.Required {
				return nil, nil, fmt.Errorf("required env var %s (mapped as %s) is not set on host", ev.From, ev.As)
			}
			continue
		}
		env = append(env, ev.As+"="+val)
	}
	return env, sprootCfg, nil
}
