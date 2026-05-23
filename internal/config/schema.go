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

// TargetConfig defines a named set of phases in a multi-target sproot.yaml.
// Extends names another target whose phases are prepended before this target's phases.
type TargetConfig struct {
	Extends string        `yaml:"extends"`
	Phases  []PhaseConfig `yaml:"phases"`
}

// SprootConfig is the top-level struct for sproot.yaml, found in the config repo.
// Either Phases or Targets must be set, not both.
// When Targets is used, sproot new --target <name> selects which target to run.
// A flat Phases list (no Targets) is treated as a single implicit default target.
type SprootConfig struct {
	SchemaVersion        int                      `yaml:"schema_version"`
	Identity             Identity                 `yaml:"identity"`
	Env                  []EnvVar                 `yaml:"env"`
	Phases               []PhaseConfig            `yaml:"phases"`
	Targets              map[string]*TargetConfig `yaml:"targets"`
	CheckpointAfterSetup bool                     `yaml:"checkpoint_after_setup"`
}

// ResolveTarget returns the phases for the named target, applying extends inheritance.
// When name is empty and Targets is defined, looks for a "default" target; if absent
// falls back to Phases. When Targets is not defined, returns Phases regardless of name.
func (c *SprootConfig) ResolveTarget(name string) ([]PhaseConfig, error) {
	if len(c.Targets) == 0 {
		return c.Phases, nil
	}
	lookup := name
	if lookup == "" {
		lookup = "default"
	}
	return resolveTargetChain(c.Targets, lookup, nil)
}

func resolveTargetChain(targets map[string]*TargetConfig, name string, visited []string) ([]PhaseConfig, error) {
	for _, seen := range visited {
		if seen == name {
			return nil, fmt.Errorf("target cycle detected: %s -> %s", visited[len(visited)-1], name)
		}
	}
	t, ok := targets[name]
	if !ok {
		return nil, fmt.Errorf("target %q not found", name)
	}
	if t.Extends == "" {
		return t.Phases, nil
	}
	parent, err := resolveTargetChain(targets, t.Extends, append(visited, name))
	if err != nil {
		return nil, err
	}
	combined := make([]PhaseConfig, 0, len(parent)+len(t.Phases))
	combined = append(combined, parent...)
	combined = append(combined, t.Phases...)
	return combined, nil
}

// Identity holds user identity fields referenced by multiple modules.
type Identity struct {
	GitUserName      string `yaml:"git_user_name"`
	GitUserEmail     string `yaml:"git_user_email"`
	GitDefaultBranch string `yaml:"git_default_branch"`
	GHUsername       string `yaml:"gh_username"`
}

// HostConfig is the struct for ~/.sproot/config.yaml, the per-machine host file.
// TokenEnv and GHTokenEnv hold environment variable *names*, not token values.
// At runtime sproot reads os.Getenv(TokenEnv) to obtain the actual token.
// SprootConfigSource is "git" (default) or "local". When "local", SprootConfigLocalPath is
// a host directory path used instead of a git clone; SprootConfigRepo and SprootConfigRef
// are not required.
type HostConfig struct {
	SprootConfigRepo      string `yaml:"sproot_config_repo"`
	SprootConfigRef       string `yaml:"sproot_config_ref"`
	SprootConfigPath      string `yaml:"sproot_config_path"` // path to config file within the repo; defaults to "sproot.yaml"
	TokenEnv              string `yaml:"token_env"`          // env var name holding the sprites API token
	GHTokenEnv            string `yaml:"gh_token_env"`       // env var name holding the GitHub PAT
	DefaultOrg            string `yaml:"default_org"`
	SprootConfigSource    string `yaml:"sproot_config_source"`     // "git" (default/empty) or "local"
	SprootConfigLocalPath string `yaml:"sproot_config_local_path"` // host directory path when sproot_config_source=local
}

// PhaseConfig represents one entry in the phases list. Type is always set.
// Exactly one typed config pointer is non-nil after unmarshaling.
type PhaseConfig struct {
	Type            string                 `yaml:"type"`
	Apt             *AptConfig             `yaml:"-"`
	UVTool          *UVToolConfig          `yaml:"-"`
	GoInstall       *GoInstallConfig       `yaml:"-"`
	CargoInstall    *CargoInstallConfig    `yaml:"-"`
	BinaryRelease   *BinaryReleaseConfig   `yaml:"-"`
	Corepack        *CorepackConfig        `yaml:"-"`
	RustComponents  *RustComponentsConfig  `yaml:"-"`
	Docker          *DockerConfig          `yaml:"-"`
	SpriteService   *SpriteServiceConfig   `yaml:"-"`
	GitIdentity     *GitIdentityConfig     `yaml:"-"`
	SSHSetup        *SSHSetupConfig        `yaml:"-"`
	GHToken         *GHTokenConfig         `yaml:"-"`
	FileTemplate    *FileTemplateConfig    `yaml:"-"`
	RCBlock         *RCBlockConfig         `yaml:"-"`
	RepoClone       *RepoCloneConfig       `yaml:"-"`
	Claude          *ClaudeConfig          `yaml:"-"`
	Npm             *NpmConfig             `yaml:"-"`
	ShellCompletion *ShellCompletionConfig `yaml:"-"`
	Cmd             *CmdConfig             `yaml:"-"`
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
		return value.Decode(p.Docker)
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
	case "claude":
		p.Claude = &ClaudeConfig{}
		return value.Decode(p.Claude)
	case "npm":
		p.Npm = &NpmConfig{}
		return value.Decode(p.Npm)
	case "shell_completion":
		p.ShellCompletion = &ShellCompletionConfig{}
		return value.Decode(p.ShellCompletion)
	case "cmd":
		p.Cmd = &CmdConfig{}
		return value.Decode(p.Cmd)
	default:
		return fmt.Errorf("unknown phase type %q", raw.Type)
	}
}

// AptSymlink is a post-install symlink created by the apt module.
type AptSymlink struct {
	From string `yaml:"from"` // source path (the installed binary, e.g. /usr/bin/batcat)
	To   string `yaml:"to"`   // symlink path to create (e.g. /usr/local/bin/bat)
}

// AptConfig installs apt packages and optionally creates post-install symlinks.
type AptConfig struct {
	Packages []string     `yaml:"packages"`
	Symlinks []AptSymlink `yaml:"symlinks"`
}

// UVTool installs a single tool via uv.
type UVTool struct {
	Name string `yaml:"name"`
	Pkg  string `yaml:"pkg"` // optional; PyPI package name when it differs from the binary name
}

// UVToolConfig installs tools via uv tool install.
// uv is installed automatically if not present.
type UVToolConfig struct {
	Tools []UVTool `yaml:"tools"`
}

// BinaryReleaseConfig downloads and installs a GitHub release asset.
// Asset supports template variables: {version}, {tag}, {tag_no_v}, {arch},
// {goos}, {dpkg_arch}, {x64_arch}, {x86_64_arch}, {arch_alias}.
// Install methods: dpkg, tar+install, raw.
// Version is an optional template for the {version} token. It defaults to the
// raw release tag; set it to "{tag_no_v}" for projects whose asset names drop
// the leading v. The download URL path always uses the raw tag regardless.
// ArchMap maps GOARCH (amd64, arm64) to a custom arch token resolved as
// {arch_alias}; use it when a project's arch naming matches none of the built-in
// vars (e.g. hadolint uses x86_64 on amd64 but arm64 on arm64).
// Checksum is an optional sha256 hex string to verify the downloaded asset.
// ChecksumAsset is an optional asset name template for a goreleaser-style
// checksums file; sproot downloads it, finds the matching line, and verifies.
// Cosign, when set, verifies a Sigstore keyless signature before install.
type BinaryReleaseConfig struct {
	Name          string            `yaml:"name"`
	Repo          string            `yaml:"repo"`
	Asset         string            `yaml:"asset"`
	Install       string            `yaml:"install"`
	Version       string            `yaml:"version"`
	ArchMap       map[string]string `yaml:"arch_map"`
	Checksum      string            `yaml:"checksum"`
	ChecksumAsset string            `yaml:"checksum_asset"`
	Cosign        *CosignConfig     `yaml:"cosign"`
}

// CosignConfig configures keyless cosign signature verification of a release
// asset (typically a checksums file). All fields are required when set.
// Blob, Signature, and Certificate are asset-name templates (same variables as
// BinaryReleaseConfig.Asset). The verification runs:
//
//	cosign verify-blob --certificate <cert> --signature <sig> \
//	  --certificate-identity-regexp <re> --certificate-oidc-issuer <issuer> <blob>
//
// Requires cosign v2+ on PATH (install it via an earlier binary_release phase).
type CosignConfig struct {
	CertificateIdentityRegexp string `yaml:"certificate_identity_regexp"`
	CertificateOIDCIssuer     string `yaml:"certificate_oidc_issuer"`
	Blob                      string `yaml:"blob"`
	Signature                 string `yaml:"signature"`
	Certificate               string `yaml:"certificate"`
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
// DaemonJSON is an optional map written to /etc/docker/daemon.json after install.
type DockerConfig struct {
	DaemonJSON map[string]any `yaml:"daemon_json"`
}

// SpriteServiceConfig registers a sprite-env managed service via the internal API socket.
// Cmd is the executable path. Args, HTTPPort, and Needs are optional.
// HTTPPort and Needs are omitted from the JSON body when zero/nil.
type SpriteServiceConfig struct {
	Service  string   `yaml:"service"`
	Cmd      string   `yaml:"cmd"`
	Args     []string `yaml:"args"`
	HTTPPort int      `yaml:"http_port"`
	Needs    []string `yaml:"needs"`
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

// RepoCloneEntry is one entry in a repo_clone repos list.
// Short form (string): "owner/repo" — cloned via SSH into base_dir/<repo>.
// Long form (map):     {url: "https://...", dest: "~/dir"} — dest is optional;
// defaults to ~/<repo-name> (last URL path component, minus .git).
type RepoCloneEntry struct {
	Raw  string // set for short string form
	URL  string // set for long map form
	Dest string // optional explicit destination (long form only)
}

// UnmarshalYAML accepts both a plain string and a {url, dest} map.
func (e *RepoCloneEntry) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		e.Raw = value.Value
		return nil
	}
	var s struct {
		URL  string `yaml:"url"`
		Dest string `yaml:"dest"`
	}
	if err := value.Decode(&s); err != nil {
		return err
	}
	e.URL = s.URL
	e.Dest = s.Dest
	return nil
}

// RepoCloneConfig clones a list of repos into configured destinations.
// base_dir is used for short "owner/repo" entries; ignored for long {url,dest} entries.
type RepoCloneConfig struct {
	BaseDir string           `yaml:"base_dir"`
	Repos   []RepoCloneEntry `yaml:"repos"`
}

// NpmConfig runs npm install in a target directory.
type NpmConfig struct {
	Dir string `yaml:"dir"`
}

// ShellCompletionEntry registers completions for one command across one or more
// shells. Gen is an optional command template (tokens {command} and {shell})
// that produces the completion script on stdout; it defaults to
// "{command} completion {shell}" (the cobra convention). The rendered template
// is split on whitespace and executed directly (no shell).
type ShellCompletionEntry struct {
	Command string   `yaml:"command"`
	Shells  []string `yaml:"shells"`
	Gen     string   `yaml:"gen"`
}

// ShellCompletionConfig generates and installs shell completion scripts for the
// listed commands into per-user completion directories (bash, zsh, fish) and
// wires zsh's fpath so it loads them.
type ShellCompletionConfig struct {
	Completions []ShellCompletionEntry `yaml:"completions"`
}

// ClaudeConfig configures Claude Code: an optional deep-merge into
// ~/.claude/settings.json, an optional `claude upgrade`, and an optional
// managed CLAUDE.md block sourced from a config-repo file.
// ClaudeMD is a config-repo-relative path (resolved like file_template's src).
// Template renders ClaudeMD as a Go template against the identity when true.
type ClaudeConfig struct {
	Settings map[string]any `yaml:"settings"`
	Upgrade  bool           `yaml:"upgrade"`
	ClaudeMD string         `yaml:"claude_md"`
	Template bool           `yaml:"template"`
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
