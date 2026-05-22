package host

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/term"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

const configSkeleton = `config_repo: ""
config_ref: main
config_path: ""       # optional; path to sproot.yaml within the config repo
token_env: SPRITES_TOKEN
gh_token_env: ""
default_org: ""
`

// RunConfigInit writes a skeleton host config file at path.
// Returns an error if the file already exists.
func RunConfigInit(path string) error {
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(expanded); err == nil {
		return fmt.Errorf("%s already exists; delete it first", expanded)
	}
	if err := os.MkdirAll(filepath.Dir(expanded), 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(expanded, []byte(configSkeleton), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	log.Stderr().Infof("wrote %s", expanded)
	return nil
}

// RunConfigInitInteractive prompts the user for each field and writes the config.
// Returns an error if the file already exists.
func RunConfigInitInteractive(path string) error {
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return err
	}
	if _, err := os.Stat(expanded); err == nil {
		return fmt.Errorf("%s already exists; delete it first", expanded)
	}

	r := bufio.NewReader(os.Stdin)

	configRepo, err := promptField(r, "config_repo (e.g. git@github.com:you/sprite.git)", "")
	if err != nil {
		return err
	}
	tokenEnv, err := promptField(r, "token_env (env var holding your sprites API token)", "SPRITES_TOKEN")
	if err != nil {
		return err
	}
	ghTokenEnv, err := promptField(r, "gh_token_env (env var holding your GitHub PAT, or blank)", "")
	if err != nil {
		return err
	}
	defaultOrg, err := promptField(r, "default_org (optional)", "")
	if err != nil {
		return err
	}

	cfg := &config.HostConfig{
		ConfigRepo: configRepo,
		ConfigRef:  "main",
		TokenEnv:   tokenEnv,
		GHTokenEnv: ghTokenEnv,
		DefaultOrg: defaultOrg,
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	content := fmt.Sprintf(
		"config_repo: %q\nconfig_ref: main\nconfig_path: \"\"\ntoken_env: %s\ngh_token_env: %s\ndefault_org: %s\n",
		configRepo, tokenEnv, ghTokenEnv, defaultOrg,
	)
	if err := os.MkdirAll(filepath.Dir(expanded), 0o750); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	if err := os.WriteFile(expanded, []byte(content), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	log.Stderr().Successf("wrote %s", expanded)
	return nil
}

// RunConfigValidate loads and validates the host config at path, reporting the result.
func RunConfigValidate(path string) error {
	l := log.Stderr()
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return err
	}
	cfg, err := config.LoadHostConfig(expanded)
	if err != nil {
		l.Errorf("load: %v", err)
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		l.Errorf("validation failed: %v", err)
		return errors.New("config validation failed")
	}
	l.Info("config is valid")
	return nil
}

// loadOrInitHostConfig loads the host config from path. If the file does not
// exist and stdin is a terminal, it offers to run the interactive init flow.
func loadOrInitHostConfig(path string) (*config.HostConfig, error) {
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return nil, err
	}

	cfg, err := config.LoadHostConfig(expanded)
	if err == nil {
		return cfg, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	// File missing: offer interactive init if stdin is a terminal.
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return nil, fmt.Errorf("no config found at %s; run 'sproot config init' to create one", expanded)
	}

	fmt.Fprintf(os.Stderr, "! no config found at %s\n", expanded)
	r := bufio.NewReader(os.Stdin)
	fmt.Fprint(os.Stderr, "  run interactive setup now? [y/N] ")
	answer, _ := r.ReadString('\n')
	if strings.ToLower(strings.TrimSpace(answer)) != "y" {
		return nil, fmt.Errorf("no config at %s; run 'sproot config init' to create one", expanded)
	}

	if err := RunConfigInitInteractive(expanded); err != nil {
		return nil, fmt.Errorf("config init: %w", err)
	}

	return config.LoadHostConfig(expanded)
}

// promptField prints a prompt to stderr and reads a trimmed line from r.
// If the user enters nothing, def is returned.
func promptField(r *bufio.Reader, label, def string) (string, error) {
	if def != "" {
		fmt.Fprintf(os.Stderr, "  %s [%s]: ", label, def)
	} else {
		fmt.Fprintf(os.Stderr, "  %s: ", label)
	}
	line, err := r.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("read input: %w", err)
	}
	v := strings.TrimSpace(line)
	if v == "" {
		return def, nil
	}
	return v, nil
}
