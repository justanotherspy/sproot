package host

import (
	"errors"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// RunValidateSprootConfig loads and validates the sproot.yaml at path.
func RunValidateSprootConfig(path string) error {
	l := log.Stderr()
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
