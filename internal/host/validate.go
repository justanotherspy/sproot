package host

import (
	"errors"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// RunValidateSprootConfig loads and validates the sproot.yaml at path.
// It also validates ~/.sproot/config.yaml when that file exists.
func RunValidateSprootConfig(path string) error {
	l := log.Stderr()

	hostPath, err := config.DefaultHostConfigPath()
	if err == nil {
		if _, statErr := os.Stat(hostPath); statErr == nil {
			hcfg, herr := config.LoadHostConfig(hostPath)
			if herr != nil {
				l.Errorf("host config: load: %v", herr)
				return herr
			}
			if herr := config.ValidateHostConfig(hcfg); herr != nil {
				l.Errorf("host config: validation failed:\n%v", herr)
				return errors.New("host config validation failed")
			}
			l.Infof("%s is valid", hostPath)
		}
	}

	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return err
	}
	cfg, err := config.LoadSprootConfig(expanded)
	if err != nil {
		l.Errorf("load: %v", err)
		return err
	}
	if err := config.ValidateSprootConfig(cfg); err != nil {
		l.Errorf("validation failed:\n%v", err)
		return errors.New("sproot.yaml validation failed")
	}
	l.Infof("%s is valid", path)
	return nil
}
