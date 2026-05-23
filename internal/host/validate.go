package host

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// RunValidate validates ~/.sproot/config.yaml (when present) and the sproot.yaml
// resolved from the configured config source: the git config repo (via the
// cached loadConfigBytes) or, when sproot_config_source=local, the local config
// directory. This mirrors what `sproot new` and `sproot push` actually use, so
// validate checks the real source of truth rather than a file in the current
// directory. When no host config exists, it falls back to ./sproot.yaml.
//
// Use RunValidateSprootConfig (the --path flag) to validate a specific local
// file instead.
func RunValidate(strict bool) error {
	l := log.Stderr()

	hcfg, present, err := validateHostConfigIfPresent(l)
	if err != nil {
		return err
	}
	if !present {
		// No host config to point at a config source; validate ./sproot.yaml as
		// a convenience (e.g. when run inside a config repo checkout).
		return validateSprootFile(l, "sproot.yaml", strict)
	}

	if hcfg.SprootConfigSource == "local" {
		path, perr := config.ExpandTilde(filepath.Join(hcfg.SprootConfigLocalPath, sprootConfigFile(hcfg.SprootConfigPath)))
		if perr != nil {
			return perr
		}
		return validateSprootFile(l, path, strict)
	}

	// git source: resolve the sproot.yaml from the configured repo.
	l.Infof("resolving sproot.yaml from %s@%s", hcfg.SprootConfigRepo, hcfg.SprootConfigRef)
	raw, lerr := loadConfigBytes(hcfg.SprootConfigRepo, hcfg.SprootConfigRef, hcfg.SprootConfigPath, l)
	if lerr != nil {
		l.Errorf("resolve sproot.yaml: %v", lerr)
		return lerr
	}
	return validateSprootBytes(l, raw, sprootConfigFile(hcfg.SprootConfigPath))
}

// RunValidateSprootConfig loads and validates the sproot.yaml at path (the
// --path flag). It also validates ~/.sproot/config.yaml when that file exists.
// When strict is false, a missing sproot.yaml is a warning rather than an error.
func RunValidateSprootConfig(path string, strict bool) error {
	l := log.Stderr()
	if _, _, err := validateHostConfigIfPresent(l); err != nil {
		return err
	}
	return validateSprootFile(l, path, strict)
}

// sprootConfigFile returns configPath when set, or the default "sproot.yaml".
func sprootConfigFile(configPath string) string {
	if configPath == "" {
		return "sproot.yaml"
	}
	return configPath
}

// validateHostConfigIfPresent validates ~/.sproot/config.yaml when it exists,
// logging success. It returns the loaded config and present=true when the file
// exists and is valid; present=false (with a nil error) when the file is absent.
func validateHostConfigIfPresent(l *log.Logger) (*config.HostConfig, bool, error) {
	hostPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return nil, false, nil
	}
	if _, statErr := os.Stat(hostPath); statErr != nil {
		return nil, false, nil
	}
	hcfg, herr := config.LoadHostConfig(hostPath)
	if herr != nil {
		l.Errorf("host config: load: %v", herr)
		return nil, false, herr
	}
	if verr := config.ValidateHostConfig(hcfg); verr != nil {
		l.Errorf("host config: validation failed:\n%v", verr)
		return nil, false, errors.New("host config validation failed")
	}
	l.Infof("%s is valid", hostPath)
	return hcfg, true, nil
}

// validateSprootFile loads and validates the sproot.yaml at a local path. A
// missing file is a warning unless strict is set.
func validateSprootFile(l *log.Logger, path string, strict bool) error {
	expanded, err := config.ExpandTilde(path)
	if err != nil {
		return err
	}
	cfg, err := config.LoadSprootConfig(expanded)
	if err != nil {
		if os.IsNotExist(err) || errors.Is(err, os.ErrNotExist) {
			if strict {
				l.Errorf("load: %v", err)
				return err
			}
			l.Warnf("%s not found (use --strict to treat this as an error)", path)
			return nil
		}
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

// validateSprootBytes loads and validates raw sproot.yaml bytes resolved from a
// config source, labeling output with name. The caller reports resolution
// failures; here a parse or validation failure is always an error.
func validateSprootBytes(l *log.Logger, raw []byte, name string) error {
	cfg, err := config.LoadSprootConfigBytes(raw)
	if err != nil {
		l.Errorf("load %s: %v", name, err)
		return err
	}
	if err := config.ValidateSprootConfig(cfg); err != nil {
		l.Errorf("validation failed:\n%v", err)
		return errors.New("sproot.yaml validation failed")
	}
	l.Infof("%s is valid", name)
	return nil
}
