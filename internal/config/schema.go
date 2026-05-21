package config

import (
	"fmt"

	"go.yaml.in/yaml/v3"
)

// EnvVar maps a host environment variable into the sprite under a given name.
// From is the variable name on the host; As is the name it gets in the sprite.
// Required causes sproot new to fail if the host variable is unset or empty.
type EnvVar struct {
	From     string `yaml:"from"`
	As       string `yaml:"as"`
	Required bool   `yaml:"required"`
}

// SprootConfig is the top-level struct for sproot.yaml, found in the config repo.
type SprootConfig struct {
	SchemaVersion      int           `yaml:"schema_version"`
	Identity           Identity      `yaml:"identity"`
	Env                []EnvVar      `yaml:"env"`
	Phases             []PhaseConfig `yaml:"phases"`
	CheckpointAfterSetup bool        `yaml:"checkpoint_after_setup"`
}

// Identity holds user identity fields referenced by multiple modules.
type Identity struct {
	GitUserName      string `yaml:"git_user_name"`
	GitUserEmail     string `yaml:"git_user_email"`
	GitDefaultBranch string `yaml:"git_default_branch"`
	GHUsername       string `yaml:"gh_username"`
}

// HostConfig is the struct for ~/.sproot/config, the per-machine host file.
// TokenEnv and GHTokenEnv hold environment variable *names*, not token values.
// At runtime sproot reads os.Getenv(TokenEnv) to obtain the actual token.
type HostConfig struct {
	ConfigRepo string `yaml:"config_repo"`
	ConfigRef  string `yaml:"config_ref"`
	ConfigPath string `yaml:"config_path"` // path to config file within the repo; defaults to "sproot.yaml"
	TokenEnv   string `yaml:"token_env"`   // env var name holding the sprites API token
	GHTokenEnv string `yaml:"gh_token_env"` // env var name holding the GitHub PAT
	DefaultOrg string `yaml:"default_org"`
}

// PhaseConfig represents one entry in the phases list. Type is always set.
// Exactly one typed config pointer is non-nil after unmarshaling.
type PhaseConfig struct {
	Type           string                `yaml:"type"`
	Apt            *AptConfig            `yaml:"-"`
	UVTool         *UVToolConfig         `yaml:"-"`
	GoInstall      *GoInstallConfig      `yaml:"-"`
	CargoInstall   *CargoInstallConfig   `yaml:"-"`
	BinaryRelease  *BinaryReleaseConfig  `yaml:"-"`
	Corepack       *CorepackConfig       `yaml:"-"`
	RustComponents *RustComponentsConfig `yaml:"-"`
	Docker         *DockerConfig         `yaml:"-"`
	SpriteService  *SpriteServiceConfig  `yaml:"-"`
	GitIdentity    *GitIdentityConfig    `yaml:"-"`
	SSHSetup       *SSHSetupConfig       `yaml:"-"`
	GHToken        *GHTokenConfig        `yaml:"-"`
	FileTemplate   *FileTemplateConfig   `yaml:"-"`
	RCBlock        *RCBlockConfig        `yaml:"-"`
	RepoClone      *RepoCloneConfig      `yaml:"-"`
	ClaudeSettings *ClaudeSettingsConfig `yaml:"-"`
	Cmd            *CmdConfig            `yaml:"-"`
}

// UnmarshalYAML decodes a phase entry using a two-pass approach: first reads
// the type field, then decodes the full node into the appropriate concrete struct.
func (p *PhaseConfig) UnmarshalYAML(value *yaml.Node) error {
	var raw struct {
		Type string `yaml:"type"`
	}
	if err := value.Decode(&raw); err != nil {
		return err
	}
	p.Type = raw.Type

	switch raw.Type {
	case "apt":
		p.Apt = &AptConfig{}
		return value.Decode(p.Apt)
	case "uv_tool":
		p.UVTool = &UVToolConfig{}
		return value.Decode(p.UVTool)
	case "go_install":
		p.GoInstall = &GoInstallConfig{}
		return value.Decode(p.GoInstall)
	case "cargo_install":
		p.CargoInstall = &CargoInstallConfig{}
		return value.Decode(p.CargoInstall)
	case "binary_release":
		p.BinaryRelease = &BinaryReleaseConfig{}
		return value.Decode(p.BinaryRelease)
	case "corepack":
		p.Corepack = &CorepackConfig{}
		return value.Decode(p.Corepack)
	case "rust_components":
		p.RustComponents = &RustComponentsConfig{}
		return value.Decode(p.RustComponents)
	case "docker":
		p.Docker = &DockerConfig{}
		return nil
	case "sprite_service":
		p.SpriteService = &SpriteServiceConfig{}
		return value.Decode(p.SpriteService)
	case "git_identity":
		p.GitIdentity = &GitIdentityConfig{}
		return value.Decode(p.GitIdentity)
	case "ssh_setup":
		p.SSHSetup = &SSHSetupConfig{}
		return nil
	case "gh_token":
		p.GHToken = &GHTokenConfig{}
		return nil
	case "file_template":
		p.FileTemplate = &FileTemplateConfig{}
		return value.Decode(p.FileTemplate)
	case "rc_block":
		p.RCBlock = &RCBlockConfig{}
		return value.Decode(p.RCBlock)
	case "repo_clone":
		p.RepoClone = &RepoCloneConfig{}
		return value.Decode(p.RepoClone)
	case "claude_settings":
		p.ClaudeSettings = &ClaudeSettingsConfig{}
		return value.Decode(p.ClaudeSettings)
	case "cmd":
		p.Cmd = &CmdConfig{}
		return value.Decode(p.Cmd)
	default:
		return fmt.Errorf("unknown phase type %q", raw.Type)
	}
}

// AptConfig installs apt packages.
type AptConfig struct {
	Packages []string `yaml:"packages"`
}

// UVTool installs a single tool via uv.
type UVTool struct {
	Name string `yaml:"name"`
}

// UVToolConfig installs tools via uv tool install.
type UVToolConfig struct {
	Tools []UVTool `yaml:"tools"`
}

// BinaryReleaseConfig downloads and installs a GitHub release asset.
// Asset supports template variables: {version}, {arch}, {goos}, {dpkg_arch}.
// Install methods: dpkg, tar+install, raw.
// Checksum is an optional sha256 hex string to verify the downloaded asset.
// ChecksumAsset is an optional asset name template for a goreleaser-style
// checksums file; sproot downloads it, finds the matching line, and verifies.
type BinaryReleaseConfig struct {
	Name          string `yaml:"name"`
	Repo          string `yaml:"repo"`
	Asset         string `yaml:"asset"`
	Install       string `yaml:"install"`
	Checksum      string `yaml:"checksum"`
	ChecksumAsset string `yaml:"checksum_asset"`
}

// CorepackConfig enables corepack and pre-activates the listed package managers.
type CorepackConfig struct {
	Managers []string `yaml:"managers"`
}

// RustComponentsConfig pins stable and installs the listed rustup components.
type RustComponentsConfig struct {
	Components []string `yaml:"components"`
}

// DockerConfig installs docker-ce via the official install script.
type DockerConfig struct{}

// SpriteServiceConfig registers a sprite-env managed service via the internal API socket.
// Cmd is the executable path (e.g. /usr/bin/dockerd). Args are optional.
type SpriteServiceConfig struct {
	Service string   `yaml:"service"`
	Cmd     string   `yaml:"cmd"`
	Args    []string `yaml:"args"`
}

// GitIdentityConfig applies git user.name, user.email, and init.defaultBranch from
// the top-level identity block. Config is an optional map of additional git config
// key-value pairs to set (e.g. pull.rebase, core.editor).
type GitIdentityConfig struct {
	Config map[string]string `yaml:"config"`
}

// SSHSetupConfig configures the injected SSH key and known_hosts.
type SSHSetupConfig struct{}

// GHTokenConfig exports GH_TOKEN from the sprite-attached app token.
type GHTokenConfig struct{}

// FileTemplateConfig copies a file from the config repo to a destination path.
// Mode is an optional octal string (e.g. "0755"). Template enables Go template
// rendering against ctx.Identity; without it the file is copied as-is.
type FileTemplateConfig struct {
	Src      string `yaml:"src"`
	Dest     string `yaml:"dest"`
	Mode     string `yaml:"mode"`
	Template bool   `yaml:"template"`
}

// RCBlockConfig writes a sentinel-delimited block to .bashrc and .zshrc.
type RCBlockConfig struct {
	Src string `yaml:"src"`
}

// RepoCloneConfig clones a list of GitHub repos into a base directory.
// Repos are specified as "owner/repo" (SSH clone from github.com is assumed).
type RepoCloneConfig struct {
	BaseDir string   `yaml:"base_dir"`
	Repos   []string `yaml:"repos"`
}

// ClaudeSettingsConfig deep-merges a JSON object into ~/.claude/settings.json.
type ClaudeSettingsConfig struct {
	Settings map[string]any `yaml:"settings"`
}

// CmdConfig runs an arbitrary command with an optional idempotency check.
// Check is a command that exits 0 if the work is already done (skips Run).
// Name is an optional display name; when set, Name() returns cmd(name).
type CmdConfig struct {
	Run   string `yaml:"run"`
	Check string `yaml:"check"`
	Name  string `yaml:"name"`
}

// GoTool is one entry in a go_install phase.
type GoTool struct {
	Pkg     string `yaml:"pkg"`
	Version string `yaml:"version"` // "latest" or a full semver like v1.2.3
}

// GoInstallConfig installs Go tools via go install pkg@version.
type GoInstallConfig struct {
	Tools []GoTool `yaml:"tools"`
}

// CargoTool is one entry in a cargo_install phase.
type CargoTool struct {
	Name     string   `yaml:"name"`
	Version  string   `yaml:"version"`  // optional; omit for latest
	Locked   bool     `yaml:"locked"`   // optional; passes --locked
	Features []string `yaml:"features"` // optional
}

// CargoInstallConfig installs Rust crates via cargo install.
type CargoInstallConfig struct {
	Tools []CargoTool `yaml:"tools"`
}
