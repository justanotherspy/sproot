package host

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

const configSkeleton = `config_repo: ""
config_ref: main
token_env: SPRITE_TOKEN
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
