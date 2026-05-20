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

	if len(cfg.Phases) == 0 {
		errs = append(errs, errors.New("phases must not be empty"))
	}

	for i, phase := range cfg.Phases {
		if phase.Type == "" {
			errs = append(errs, fmt.Errorf("phases[%d].type is required", i))
			continue
		}
		if !knownPhaseTypes[phase.Type] {
			errs = append(errs, fmt.Errorf("phases[%d].type %q is not a known module type", i, phase.Type))
			continue
		}
		switch phase.Type {
		case "corepack":
			if c := phase.Corepack; c != nil && len(c.Managers) == 0 {
				errs = append(errs, fmt.Errorf("phases[%d] (corepack): managers must not be empty", i))
			}
		case "rust_components":
			if r := phase.RustComponents; r != nil && len(r.Components) == 0 {
				errs = append(errs, fmt.Errorf("phases[%d] (rust_components): components must not be empty", i))
			}
		case "go_install":
			if gi := phase.GoInstall; gi != nil && len(gi.Tools) == 0 {
				errs = append(errs, fmt.Errorf("phases[%d] (go_install): tools must not be empty", i))
			}
		case "cargo_install":
			if ci := phase.CargoInstall; ci != nil && len(ci.Tools) == 0 {
				errs = append(errs, fmt.Errorf("phases[%d] (cargo_install): tools must not be empty", i))
			}
		case "sprite_service":
			if ss := phase.SpriteService; ss != nil && ss.Cmd == "" {
				errs = append(errs, fmt.Errorf("phases[%d] (sprite_service): cmd is required", i))
			}
		case "binary_release":
			if br := phase.BinaryRelease; br != nil {
				if br.Name == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (binary_release): name is required", i))
				}
				if br.Repo == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (binary_release): repo is required", i))
				}
				if br.Asset == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (binary_release): asset is required", i))
				}
			}
		case "file_template":
			if ft := phase.FileTemplate; ft != nil {
				if ft.Src == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (file_template): src is required", i))
				}
				if ft.Dest == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (file_template): dest is required", i))
				}
			}
		case "rc_block":
			if rb := phase.RCBlock; rb != nil && rb.Src == "" {
				errs = append(errs, fmt.Errorf("phases[%d] (rc_block): src is required", i))
			}
		case "repo_clone":
			if rc := phase.RepoClone; rc != nil {
				if rc.BaseDir == "" {
					errs = append(errs, fmt.Errorf("phases[%d] (repo_clone): base_dir is required", i))
				}
				if len(rc.Repos) == 0 {
					errs = append(errs, fmt.Errorf("phases[%d] (repo_clone): repos must not be empty", i))
				}
			}
		case "cmd":
			if c := phase.Cmd; c != nil && c.Run == "" {
				errs = append(errs, fmt.Errorf("phases[%d] (cmd): run is required", i))
			}
		}
	}

	return errors.Join(errs...)
}

// ValidateHostConfig validates a parsed HostConfig.
func ValidateHostConfig(cfg *HostConfig) error {
	var errs []error
	if cfg.ConfigRepo == "" {
		errs = append(errs, errors.New("config_repo is required"))
	}
	if cfg.ConfigRef == "" {
		errs = append(errs, errors.New("config_ref is required"))
	}
	if cfg.TokenEnv == "" {
		errs = append(errs, errors.New("token_env is required"))
	}
	if cfg.GHTokenEnv == "" {
		errs = append(errs, errors.New("gh_token_env is required"))
	}
	return errors.Join(errs...)
}
