package config

import (
	"errors"
	"fmt"
)

// knownPhaseTypes is the full set of module types defined in Phase 3.
var knownPhaseTypes = map[string]bool{
	"apt":             true,
	"uv_tool":         true,
	"go_install":      true,
	"cargo_install":   true,
	"binary_release":  true,
	"corepack":        true,
	"rust_components": true,
	"docker":          true,
	"sprite_service":  true,
	"git_identity":    true,
	"ssh_setup":       true,
	"gh_token":        true,
	"file_template":   true,
	"rc_block":        true,
	"repo_clone":      true,
	"claude_settings": true,
	"cmd":             true,
}

// ValidateSprootConfig validates a parsed SprootConfig. It collects all
// violations and returns them together so callers see the full picture.
func ValidateSprootConfig(cfg *SprootConfig) error {
	var errs []error

	if cfg.SchemaVersion == 0 {
		errs = append(errs, errors.New("schema_version is required"))
	} else if cfg.SchemaVersion != 1 {
		errs = append(errs, fmt.Errorf("schema_version %d is not supported (expected 1)", cfg.SchemaVersion))
	}

	if cfg.Identity.GitUserName == "" {
		errs = append(errs, errors.New("identity.git_user_name is required"))
	}
	if cfg.Identity.GitUserEmail == "" {
		errs = append(errs, errors.New("identity.git_user_email is required"))
	}
	if cfg.Identity.GitDefaultBranch == "" {
		errs = append(errs, errors.New("identity.git_default_branch is required"))
	}
	if cfg.Identity.GHUsername == "" {
		errs = append(errs, errors.New("identity.gh_username is required"))
	}

	for i, ev := range cfg.Env {
		if ev.From == "" {
			errs = append(errs, fmt.Errorf("env[%d].from is required", i))
		}
		if ev.As == "" {
			errs = append(errs, fmt.Errorf("env[%d].as is required", i))
		}
	}

	hasFlat := len(cfg.Phases) > 0
	hasTargets := len(cfg.Targets) > 0

	if !hasFlat && !hasTargets {
		errs = append(errs, errors.New("phases (or targets) must not be empty"))
	}
	if hasFlat && hasTargets {
		errs = append(errs, errors.New("phases and targets cannot both be set; use one or the other"))
	}

	if hasTargets {
		for tname, t := range cfg.Targets {
			if t == nil {
				errs = append(errs, fmt.Errorf("target %q is nil", tname))
				continue
			}
			if len(t.Phases) == 0 && t.Extends == "" {
				errs = append(errs, fmt.Errorf("target %q: phases must not be empty (or set extends)", tname))
			}
			if t.Extends != "" {
				if _, ok := cfg.Targets[t.Extends]; !ok {
					errs = append(errs, fmt.Errorf("target %q: extends %q not found", tname, t.Extends))
				}
			}
			for i, phase := range t.Phases {
				errs = append(errs, validatePhase(fmt.Sprintf("target[%s].phases[%d]", tname, i), phase)...)
			}
		}
		// Cycle detection across all targets.
		for tname := range cfg.Targets {
			if _, err := cfg.ResolveTarget(tname); err != nil {
				errs = append(errs, fmt.Errorf("target %q: %w", tname, err))
			}
		}
	}

	for i, phase := range cfg.Phases {
		errs = append(errs, validatePhase(fmt.Sprintf("phases[%d]", i), phase)...)
	}

	return errors.Join(errs...)
}

// validatePhase validates a single PhaseConfig entry. prefix is used in error messages (e.g. "phases[0]").
func validatePhase(prefix string, phase PhaseConfig) []error {
	var errs []error
	if phase.Type == "" {
		return []error{fmt.Errorf("%s.type is required", prefix)}
	}
	if !knownPhaseTypes[phase.Type] {
		return []error{fmt.Errorf("%s.type %q is not a known module type", prefix, phase.Type)}
	}
	switch phase.Type {
	case "corepack":
		if c := phase.Corepack; c != nil && len(c.Managers) == 0 {
			errs = append(errs, fmt.Errorf("%s (corepack): managers must not be empty", prefix))
		}
	case "rust_components":
		if r := phase.RustComponents; r != nil && len(r.Components) == 0 {
			errs = append(errs, fmt.Errorf("%s (rust_components): components must not be empty", prefix))
		}
	case "go_install":
		if gi := phase.GoInstall; gi != nil && len(gi.Tools) == 0 {
			errs = append(errs, fmt.Errorf("%s (go_install): tools must not be empty", prefix))
		}
	case "cargo_install":
		if ci := phase.CargoInstall; ci != nil && len(ci.Tools) == 0 {
			errs = append(errs, fmt.Errorf("%s (cargo_install): tools must not be empty", prefix))
		}
	case "sprite_service":
		if ss := phase.SpriteService; ss != nil && ss.Cmd == "" {
			errs = append(errs, fmt.Errorf("%s (sprite_service): cmd is required", prefix))
		}
	case "binary_release":
		if br := phase.BinaryRelease; br != nil {
			if br.Name == "" {
				errs = append(errs, fmt.Errorf("%s (binary_release): name is required", prefix))
			}
			if br.Repo == "" {
				errs = append(errs, fmt.Errorf("%s (binary_release): repo is required", prefix))
			}
			if br.Asset == "" {
				errs = append(errs, fmt.Errorf("%s (binary_release): asset is required", prefix))
			}
		}
	case "file_template":
		if ft := phase.FileTemplate; ft != nil {
			if ft.Src == "" {
				errs = append(errs, fmt.Errorf("%s (file_template): src is required", prefix))
			}
			if ft.Dest == "" {
				errs = append(errs, fmt.Errorf("%s (file_template): dest is required", prefix))
			}
		}
	case "rc_block":
		if rb := phase.RCBlock; rb != nil && rb.Src == "" {
			errs = append(errs, fmt.Errorf("%s (rc_block): src is required", prefix))
		}
	case "repo_clone":
		if rc := phase.RepoClone; rc != nil {
			if rc.BaseDir == "" {
				errs = append(errs, fmt.Errorf("%s (repo_clone): base_dir is required", prefix))
			}
			if len(rc.Repos) == 0 {
				errs = append(errs, fmt.Errorf("%s (repo_clone): repos must not be empty", prefix))
			}
		}
	case "cmd":
		if c := phase.Cmd; c != nil && c.Run == "" {
			errs = append(errs, fmt.Errorf("%s (cmd): run is required", prefix))
		}
	}
	return errs
}

// ValidateHostConfig validates a parsed HostConfig.
func ValidateHostConfig(cfg *HostConfig) error {
	var errs []error
	switch cfg.SprootConfigSource {
	case "", "git":
		if cfg.SprootConfigRepo == "" {
			errs = append(errs, errors.New("sproot_config_repo is required"))
		}
		if cfg.SprootConfigRef == "" {
			errs = append(errs, errors.New("sproot_config_ref is required"))
		}
	case "local":
		if cfg.SprootConfigLocalPath == "" {
			errs = append(errs, errors.New("sproot_config_local_path is required when sproot_config_source=local"))
		}
	default:
		errs = append(errs, fmt.Errorf("sproot_config_source %q is not valid (use \"git\" or \"local\")", cfg.SprootConfigSource))
	}
	if cfg.TokenEnv == "" {
		errs = append(errs, errors.New("token_env is required"))
	}
	return errors.Join(errs...)
}
