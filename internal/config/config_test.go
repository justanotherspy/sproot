package config

import (
	"os"
	"strings"
	"testing"
)

func testdataPath(name string) string {
	return "testdata/" + name
}

// --- SprootConfig happy path ---

func TestLoadSprootConfig_HappyPath(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SchemaVersion != 1 {
		t.Errorf("schema_version: got %d, want 1", cfg.SchemaVersion)
	}
	if cfg.Identity.GitUserName != "Daniel Schwartz" {
		t.Errorf("git_user_name: got %q", cfg.Identity.GitUserName)
	}
	if cfg.Identity.GitUserEmail != "danielschwar@gmail.com" {
		t.Errorf("git_user_email: got %q", cfg.Identity.GitUserEmail)
	}
	if cfg.Identity.GitDefaultBranch != "main" {
		t.Errorf("git_default_branch: got %q", cfg.Identity.GitDefaultBranch)
	}
	if cfg.Identity.GHUsername != "justanotherspy" {
		t.Errorf("gh_username: got %q", cfg.Identity.GHUsername)
	}
	if len(cfg.Phases) != 18 {
		t.Fatalf("phases: got %d, want 18", len(cfg.Phases))
	}
	if cfg.Phases[0].Type != "apt" {
		t.Errorf("phases[0].type: got %q, want apt", cfg.Phases[0].Type)
	}
}

func TestSprootConfig_EnvBlock(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Env) != 1 {
		t.Fatalf("env: got %d entries, want 1", len(cfg.Env))
	}
	ev := cfg.Env[0]
	if ev.From != "MY_GH_TOKEN" {
		t.Errorf("env[0].from: got %q", ev.From)
	}
	if ev.As != "GH_TOKEN" {
		t.Errorf("env[0].as: got %q", ev.As)
	}
	if !ev.Required {
		t.Error("env[0].required: got false, want true")
	}
}

func TestPhaseConfig_BinaryReleaseNewFields(t *testing.T) {
	yaml := `schema_version: 1
identity:
  git_user_name: a
  git_user_email: b@c.d
  git_default_branch: main
  gh_username: x
phases:
  - type: binary_release
    name: trufflehog
    repo: trufflesecurity/trufflehog
    version: "{tag_no_v}"
    asset: "trufflehog_{version}_linux_{arch}.tar.gz"
    arch_map:
      amd64: x86_64
      arm64: arm64
    checksum_asset: "trufflehog_{version}_checksums.txt"
    install: tar+install
    cosign:
      blob: "trufflehog_{version}_checksums.txt"
      signature: "trufflehog_{version}_checksums.txt.sig"
      certificate: "trufflehog_{version}_checksums.txt.pem"
      certificate_oidc_issuer: "https://token.actions.githubusercontent.com"
      certificate_identity_regexp: "https://github.com/trufflesecurity/.*"
  - type: claude
    upgrade: true
    settings:
      theme: dark
    claude_md: files/CLAUDE.md
`
	dir := t.TempDir()
	path := dir + "/sproot.yaml"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSprootConfig(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := ValidateSprootConfig(cfg); err != nil {
		t.Fatalf("validate: %v", err)
	}
	br := cfg.Phases[0].BinaryRelease
	if br == nil {
		t.Fatal("BinaryRelease is nil")
	}
	if br.Version != "{tag_no_v}" {
		t.Errorf("version: got %q", br.Version)
	}
	if br.ArchMap["amd64"] != "x86_64" || br.ArchMap["arm64"] != "arm64" {
		t.Errorf("arch_map: got %v", br.ArchMap)
	}
	if br.Cosign == nil || br.Cosign.Blob == "" || br.Cosign.Signature == "" || br.Cosign.Certificate == "" {
		t.Errorf("cosign: got %+v", br.Cosign)
	}
	cl := cfg.Phases[1].Claude
	if cl == nil || !cl.Upgrade || cl.ClaudeMD != "files/CLAUDE.md" || cl.Settings["theme"] != "dark" {
		t.Errorf("claude: got %+v", cl)
	}
}

func TestPhaseConfig_AptFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[0]
	if p.Apt == nil {
		t.Fatal("Apt is nil")
	}
	want := []string{"shellcheck", "jq"}
	if len(p.Apt.Packages) != len(want) {
		t.Fatalf("packages: got %v, want %v", p.Apt.Packages, want)
	}
	for i, pkg := range want {
		if p.Apt.Packages[i] != pkg {
			t.Errorf("packages[%d]: got %q, want %q", i, p.Apt.Packages[i], pkg)
		}
	}
}

func TestPhaseConfig_UVToolFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[1]
	if p.UVTool == nil {
		t.Fatal("UVTool is nil")
	}
	if len(p.UVTool.Tools) != 2 {
		t.Fatalf("tools: got %d, want 2", len(p.UVTool.Tools))
	}
	if p.UVTool.Tools[0].Name != "ruff" {
		t.Errorf("tools[0].name: got %q, want ruff", p.UVTool.Tools[0].Name)
	}
	if p.UVTool.Tools[1].Name != "pre-commit" {
		t.Errorf("tools[1].name: got %q, want pre-commit", p.UVTool.Tools[1].Name)
	}
}

func TestPhaseConfig_BinaryReleaseFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[2]
	if p.BinaryRelease == nil {
		t.Fatal("BinaryRelease is nil")
	}
	br := p.BinaryRelease
	if br.Name != "cosign" {
		t.Errorf("name: got %q", br.Name)
	}
	if br.Repo != "sigstore/cosign" {
		t.Errorf("repo: got %q", br.Repo)
	}
	if br.Asset != "cosign_{version}_{arch}.deb" {
		t.Errorf("asset: got %q", br.Asset)
	}
	if br.Install != "dpkg" {
		t.Errorf("install: got %q", br.Install)
	}
}

func TestPhaseConfig_FileTemplateFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[3]
	if p.FileTemplate == nil {
		t.Fatal("FileTemplate is nil")
	}
	ft := p.FileTemplate
	if ft.Src != "files/statusline.py" {
		t.Errorf("src: got %q", ft.Src)
	}
	if ft.Dest != "~/.claude/statusline.py" {
		t.Errorf("dest: got %q", ft.Dest)
	}
	if ft.Mode != "0755" {
		t.Errorf("mode: got %q", ft.Mode)
	}
}

func TestPhaseConfig_RCBlockFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[4]
	if p.RCBlock == nil {
		t.Fatal("RCBlock is nil")
	}
	if p.RCBlock.Src != "files/rc_additions.sh" {
		t.Errorf("src: got %q", p.RCBlock.Src)
	}
}

func TestPhaseConfig_RepoCloneFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[5]
	if p.RepoClone == nil {
		t.Fatal("RepoClone is nil")
	}
	rc := p.RepoClone
	if rc.BaseDir != "~/repos" {
		t.Errorf("base_dir: got %q", rc.BaseDir)
	}
	if len(rc.Repos) != 2 {
		t.Fatalf("repos: got %d, want 2", len(rc.Repos))
	}
	if rc.Repos[0].Raw != "justanotherspy/garlic" {
		t.Errorf("repos[0]: got %q", rc.Repos[0].Raw)
	}
}

func TestPhaseConfig_GoInstallFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[9]
	if p.GoInstall == nil {
		t.Fatal("GoInstall is nil")
	}
	if len(p.GoInstall.Tools) != 1 {
		t.Fatalf("tools: got %d, want 1", len(p.GoInstall.Tools))
	}
	if p.GoInstall.Tools[0].Pkg != "golang.org/x/tools/cmd/goimports" {
		t.Errorf("pkg: got %q", p.GoInstall.Tools[0].Pkg)
	}
	if p.GoInstall.Tools[0].Version != "latest" {
		t.Errorf("version: got %q", p.GoInstall.Tools[0].Version)
	}
}

func TestPhaseConfig_CargoInstallFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[10]
	if p.CargoInstall == nil {
		t.Fatal("CargoInstall is nil")
	}
	if len(p.CargoInstall.Tools) != 2 {
		t.Fatalf("tools: got %d, want 2", len(p.CargoInstall.Tools))
	}
	if p.CargoInstall.Tools[0].Name != "ripgrep" {
		t.Errorf("tools[0].name: got %q", p.CargoInstall.Tools[0].Name)
	}
	if p.CargoInstall.Tools[1].Name != "cargo-nextest" {
		t.Errorf("tools[1].name: got %q", p.CargoInstall.Tools[1].Name)
	}
	if p.CargoInstall.Tools[1].Version != "0.9.72" {
		t.Errorf("tools[1].version: got %q", p.CargoInstall.Tools[1].Version)
	}
	if !p.CargoInstall.Tools[1].Locked {
		t.Error("tools[1].locked: got false, want true")
	}
}

func TestPhaseConfig_CmdFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[16]
	if p.Cmd == nil {
		t.Fatal("Cmd is nil")
	}
	if p.Cmd.Run != "echo hello" {
		t.Errorf("run: got %q", p.Cmd.Run)
	}
	if p.Cmd.Check != "true" {
		t.Errorf("check: got %q", p.Cmd.Check)
	}
}

func TestPhaseConfig_NixFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[17]
	if p.Nix == nil {
		t.Fatal("Nix is nil")
	}
	if len(p.Nix.Packages) != 3 {
		t.Fatalf("packages: got %d, want 3", len(p.Nix.Packages))
	}
	// Short string form.
	if got := p.Nix.Packages[0]; got.Name != "hello" || got.FlakeRef() != "nixpkgs#hello" || got.BinName() != "hello" {
		t.Errorf("packages[0]: got %+v", got)
	}
	// Long form with bin override.
	if got := p.Nix.Packages[1]; got.Name != "ripgrep" || got.BinName() != "rg" {
		t.Errorf("packages[1]: got %+v", got)
	}
	// Long form with explicit flake ref.
	if got := p.Nix.Packages[2]; got.FlakeRef() != "nixpkgs#nixfmt-rfc-style" || got.BinName() != "nixfmt" {
		t.Errorf("packages[2]: got %+v", got)
	}
	if p.Nix.SetupScript != "files/nix-setup.sh" {
		t.Errorf("setup_script: got %q", p.Nix.SetupScript)
	}
	if !p.Nix.DaemonServiceEnabled() {
		t.Error("daemon_service: got disabled, want enabled")
	}
}

func TestPhaseConfig_CorepackFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[6]
	if p.Corepack == nil {
		t.Fatal("Corepack is nil")
	}
	want := []string{"pnpm", "yarn"}
	if len(p.Corepack.Managers) != len(want) {
		t.Fatalf("managers: got %v, want %v", p.Corepack.Managers, want)
	}
	for i, m := range want {
		if p.Corepack.Managers[i] != m {
			t.Errorf("managers[%d]: got %q, want %q", i, p.Corepack.Managers[i], m)
		}
	}
}

func TestPhaseConfig_RustComponentsFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[7]
	if p.RustComponents == nil {
		t.Fatal("RustComponents is nil")
	}
	want := []string{"clippy", "rustfmt", "rust-analyzer"}
	if len(p.RustComponents.Components) != len(want) {
		t.Fatalf("components: got %v, want %v", p.RustComponents.Components, want)
	}
	for i, c := range want {
		if p.RustComponents.Components[i] != c {
			t.Errorf("components[%d]: got %q, want %q", i, p.RustComponents.Components[i], c)
		}
	}
}

func TestPhaseConfig_EmptyStructTypes(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		idx  int
		name string
		fn   func(PhaseConfig) bool
	}{
		{8, "docker", func(p PhaseConfig) bool { return p.Docker != nil }},
		{12, "git_identity", func(p PhaseConfig) bool { return p.GitIdentity != nil }},
		{13, "ssh_setup", func(p PhaseConfig) bool { return p.SSHSetup != nil }},
		{14, "gh_token", func(p PhaseConfig) bool { return p.GHToken != nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			p := cfg.Phases[tc.idx]
			if p.Type != tc.name {
				t.Errorf("type: got %q, want %q", p.Type, tc.name)
			}
			if !tc.fn(p) {
				t.Errorf("typed config pointer is nil for %q", tc.name)
			}
		})
	}
}

// --- HostConfig happy path ---

func TestLoadHostConfig_HappyPath(t *testing.T) {
	cfg, err := LoadHostConfig(testdataPath("host_config.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.SprootConfigRepo != "git@github.com:justanotherspy/sprite.git" {
		t.Errorf("sproot_config_repo: got %q", cfg.SprootConfigRepo)
	}
	if cfg.SprootConfigRef != "main" {
		t.Errorf("sproot_config_ref: got %q", cfg.SprootConfigRef)
	}
	if cfg.TokenEnv != "SPRITES_TOKEN" {
		t.Errorf("token_env: got %q", cfg.TokenEnv)
	}
	if cfg.GHTokenEnv != "GITHUB_TOKEN" {
		t.Errorf("gh_token_env: got %q", cfg.GHTokenEnv)
	}
	if cfg.DefaultOrg != "" {
		t.Errorf("default_org: got %q, want empty", cfg.DefaultOrg)
	}
}

// --- Load errors ---

func TestLoadSprootConfig_FileNotFound(t *testing.T) {
	_, err := LoadSprootConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading sproot.yaml") {
		t.Errorf("error %q does not contain expected substring", err.Error())
	}
}

func TestLoadHostConfig_FileNotFound(t *testing.T) {
	_, err := LoadHostConfig("testdata/nonexistent.yaml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "reading host config") {
		t.Errorf("error %q does not contain expected substring", err.Error())
	}
}

func TestLoadSprootConfig_InvalidYAML(t *testing.T) {
	// Write a temp file with invalid YAML via a known invalid phase type
	// (the invalid YAML path uses a separate fixture approach).
	// Instead, use a raw string that is syntactically broken.
	// We test this via a temp file written inline.
	t.TempDir() // ensure temp support works

	dir := t.TempDir()
	path := dir + "/bad.yaml"

	if err := writeFile(t, path, `{{{`); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSprootConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "parsing sproot.yaml") {
		t.Errorf("error %q does not contain expected substring", err.Error())
	}
}

func TestLoadSprootConfig_UnknownType(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/unknown.yaml"

	yaml := `
schema_version: 1
identity:
  git_user_name: Test
  git_user_email: test@example.com
  git_default_branch: main
  gh_username: testuser
phases:
  - type: bogus
`
	if err := writeFile(t, path, yaml); err != nil {
		t.Fatal(err)
	}
	_, err := LoadSprootConfig(path)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `unknown phase type "bogus"`) {
		t.Errorf("error %q does not contain expected substring", err.Error())
	}
}

// --- ValidateSprootConfig failures ---

func TestValidateSprootConfig_Errors(t *testing.T) {
	base := func() *SprootConfig {
		return &SprootConfig{
			SchemaVersion: 1,
			Identity: Identity{
				GitUserName:      "Test User",
				GitUserEmail:     "test@example.com",
				GitDefaultBranch: "main",
				GHUsername:       "testuser",
			},
			Phases: []PhaseConfig{
				{Type: "apt", Apt: &AptConfig{Packages: []string{"curl"}}},
			},
		}
	}

	cases := []struct {
		name    string
		mutate  func(*SprootConfig)
		wantErr string
	}{
		{
			"env_missing_from",
			func(c *SprootConfig) { c.Env = []EnvVar{{As: "GH_TOKEN", Required: true}} },
			"env[0].from is required",
		},
		{
			"env_missing_as",
			func(c *SprootConfig) { c.Env = []EnvVar{{From: "MY_GH_TOKEN"}} },
			"env[0].as is required",
		},
		{
			"missing_schema_version",
			func(c *SprootConfig) { c.SchemaVersion = 0 },
			"schema_version is required",
		},
		{
			"unsupported_schema_version",
			func(c *SprootConfig) { c.SchemaVersion = 99 },
			"schema_version 99 is not supported",
		},
		{
			"missing_git_user_name",
			func(c *SprootConfig) { c.Identity.GitUserName = "" },
			"identity.git_user_name is required",
		},
		{
			"missing_git_user_email",
			func(c *SprootConfig) { c.Identity.GitUserEmail = "" },
			"identity.git_user_email is required",
		},
		{
			"missing_git_default_branch",
			func(c *SprootConfig) { c.Identity.GitDefaultBranch = "" },
			"identity.git_default_branch is required",
		},
		{
			"missing_gh_username",
			func(c *SprootConfig) { c.Identity.GHUsername = "" },
			"identity.gh_username is required",
		},
		{
			"empty_phases",
			func(c *SprootConfig) { c.Phases = nil },
			"phases (or targets) must not be empty",
		},
		{
			"binary_release_missing_name",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{Repo: "x/y", Asset: "foo"}}}
			},
			"phases[0] (binary_release): name is required",
		},
		{
			"binary_release_missing_repo",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{Name: "tool", Asset: "foo"}}}
			},
			"phases[0] (binary_release): repo is required",
		},
		{
			"binary_release_missing_asset",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{Name: "tool", Repo: "x/y"}}}
			},
			"phases[0] (binary_release): asset is required",
		},
		{
			"file_template_missing_src",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "file_template", FileTemplate: &FileTemplateConfig{Dest: "/tmp/x"}}}
			},
			"phases[0] (file_template): src is required",
		},
		{
			"file_template_missing_dest",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "file_template", FileTemplate: &FileTemplateConfig{Src: "files/x"}}}
			},
			"phases[0] (file_template): dest is required",
		},
		{
			"rc_block_missing_src",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "rc_block", RCBlock: &RCBlockConfig{}}}
			},
			"phases[0] (rc_block): src is required",
		},
		{
			"repo_clone_missing_base_dir",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "repo_clone", RepoClone: &RepoCloneConfig{Repos: []RepoCloneEntry{{Raw: "a/b"}}}}}
			},
			"phases[0] (repo_clone): base_dir is required",
		},
		{
			"repo_clone_empty_repos",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "repo_clone", RepoClone: &RepoCloneConfig{BaseDir: "~/repos"}}}
			},
			"phases[0] (repo_clone): repos must not be empty",
		},
		{
			"corepack_empty_managers",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "corepack", Corepack: &CorepackConfig{}}}
			},
			"phases[0] (corepack): managers must not be empty",
		},
		{
			"rust_components_empty_components",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "rust_components", RustComponents: &RustComponentsConfig{}}}
			},
			"phases[0] (rust_components): components must not be empty",
		},
		{
			"go_install_empty_tools",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "go_install", GoInstall: &GoInstallConfig{}}}
			},
			"phases[0] (go_install): tools must not be empty",
		},
		{
			"cargo_install_empty_tools",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "cargo_install", CargoInstall: &CargoInstallConfig{}}}
			},
			"phases[0] (cargo_install): tools must not be empty",
		},
		{
			"sprite_service_missing_cmd",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "sprite_service", SpriteService: &SpriteServiceConfig{Service: "dockerd"}}}
			},
			"phases[0] (sprite_service): cmd is required",
		},
		{
			"cmd_missing_run",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "cmd", Cmd: &CmdConfig{}}}
			},
			"phases[0] (cmd): run is required",
		},
		{
			"binary_release_arch_alias_without_map",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{
					Name: "hadolint", Repo: "h/h", Asset: "hadolint-linux-{arch_alias}",
				}}}
			},
			"asset uses {arch_alias} but arch_map is not set",
		},
		{
			"binary_release_cosign_missing_blob",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{
					Name: "t", Repo: "t/t", Asset: "t.tar.gz",
					Cosign: &CosignConfig{
						CertificateIdentityRegexp: "x", CertificateOIDCIssuer: "y",
						Signature: "s", Certificate: "c",
					},
				}}}
			},
			"binary_release.cosign): blob is required",
		},
		{
			"claude_nothing_set",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "claude", Claude: &ClaudeConfig{}}}
			},
			"set at least one of settings, upgrade, or claude_md",
		},
		{
			"claude_settings_is_now_unknown",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "claude_settings"}}
			},
			`"claude_settings" is not a known module type`,
		},
		{
			"apt_empty",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "apt", Apt: &AptConfig{}}}
			},
			"phases[0] (apt): packages or symlinks must not be empty",
		},
		{
			"uv_tool_empty",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "uv_tool", UVTool: &UVToolConfig{}}}
			},
			"phases[0] (uv_tool): tools must not be empty",
		},
		{
			"npm_missing_dir",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "npm", Npm: &NpmConfig{}}}
			},
			"phases[0] (npm): dir is required",
		},
		{
			"binary_release_invalid_install",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{
					Name: "tool", Repo: "x/y", Asset: "foo", Install: "bogus",
				}}}
			},
			`phases[0] (binary_release): install "bogus" is not valid`,
		},
		{
			"binary_release_empty_install",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "binary_release", BinaryRelease: &BinaryReleaseConfig{
					Name: "tool", Repo: "x/y", Asset: "foo",
				}}}
			},
			`phases[0] (binary_release): install "" is not valid`,
		},
		{
			"file_template_bad_mode",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "file_template", FileTemplate: &FileTemplateConfig{
					Src: "files/x", Dest: "/tmp/x", Mode: "98a",
				}}}
			},
			`phases[0] (file_template): mode "98a" is not a valid octal file mode`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(cfg)
			err := ValidateSprootConfig(cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateSprootConfig_MultipleErrors(t *testing.T) {
	cfg := &SprootConfig{
		SchemaVersion: 1,
		Identity: Identity{
			GitDefaultBranch: "main",
			GHUsername:       "testuser",
			// GitUserName and GitUserEmail intentionally missing
		},
		Phases: []PhaseConfig{
			{Type: "apt", Apt: &AptConfig{Packages: []string{"curl"}}},
		},
	}
	err := ValidateSprootConfig(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "identity.git_user_name is required") {
		t.Errorf("error missing git_user_name message: %q", msg)
	}
	if !strings.Contains(msg, "identity.git_user_email is required") {
		t.Errorf("error missing git_user_email message: %q", msg)
	}
}

// --- ValidateHostConfig failures ---

func TestValidateHostConfig_Errors(t *testing.T) {
	base := func() *HostConfig {
		return &HostConfig{
			SprootConfigRepo: "git@github.com:user/repo.git",
			SprootConfigRef:  "main",
			TokenEnv:         "SPRITES_TOKEN",
			GHTokenEnv:       "GITHUB_TOKEN",
		}
	}

	cases := []struct {
		name    string
		mutate  func(*HostConfig)
		wantErr string
	}{
		{"missing_config_repo", func(c *HostConfig) { c.SprootConfigRepo = "" }, "sproot_config_repo is required"},
		{"missing_config_ref", func(c *HostConfig) { c.SprootConfigRef = "" }, "sproot_config_ref is required"},
		{"missing_token_env", func(c *HostConfig) { c.TokenEnv = "" }, "token_env is required"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base()
			tc.mutate(cfg)
			err := ValidateHostConfig(cfg)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.wantErr)
			}
		})
	}
}

func TestValidateHostConfig_EmptyGHTokenEnvIsValid(t *testing.T) {
	cfg := &HostConfig{
		SprootConfigRepo: "git@github.com:user/repo.git",
		SprootConfigRef:  "main",
		TokenEnv:         "SPRITES_TOKEN",
		GHTokenEnv:       "",
	}
	if err := ValidateHostConfig(cfg); err != nil {
		t.Errorf("expected no error with empty gh_token_env, got: %v", err)
	}
}

func TestLoadHostConfig_TokenEnvDefault(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/config"
	if err := writeFile(t, path, "sproot_config_repo: git@github.com:u/r.git\nsproot_config_ref: main\n"); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadHostConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.TokenEnv != "SPRITES_TOKEN" {
		t.Errorf("TokenEnv default: got %q, want SPRITES_TOKEN", cfg.TokenEnv)
	}
}

// --- Utilities ---

func TestExpandTilde_NoTilde(t *testing.T) {
	path := "/absolute/path/to/file"
	got, err := ExpandTilde(path)
	if err != nil {
		t.Fatal(err)
	}
	if got != path {
		t.Errorf("got %q, want %q", got, path)
	}
}

func TestExpandTilde_WithTilde(t *testing.T) {
	got, err := ExpandTilde("~/foo/bar")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(got, "/foo/bar") {
		t.Errorf("got %q, expected to end with /foo/bar", got)
	}
	if strings.HasPrefix(got, "~") {
		t.Errorf("got %q, tilde was not expanded", got)
	}
}

func TestExpandTilde_TildeOnly(t *testing.T) {
	got, err := ExpandTilde("~")
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(got, "~") {
		t.Errorf("got %q, tilde was not expanded", got)
	}
	if got == "" {
		t.Error("got empty string")
	}
}

func TestDefaultHostConfigPath(t *testing.T) {
	path, err := DefaultHostConfigPath()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, ".sproot/config.yaml") && !strings.HasSuffix(path, ".sproot/config") {
		t.Errorf("path %q does not end with .sproot/config.yaml or .sproot/config", path)
	}
}

// --- Multi-target fixture ---

func TestLoadSprootConfig_Targets(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot_targets.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Targets) != 2 {
		t.Fatalf("targets: got %d, want 2", len(cfg.Targets))
	}
	if err := ValidateSprootConfig(cfg); err != nil {
		t.Errorf("validate failed: %v", err)
	}
	// Base target: 2 phases.
	base := cfg.Targets["base"]
	if base == nil || len(base.Phases) != 2 {
		t.Errorf("base target: got %+v, want 2 phases", base)
	}
	// Web target extends base: ResolveTarget returns 4 phases.
	phases, err := cfg.ResolveTarget("web")
	if err != nil {
		t.Fatalf("ResolveTarget web: %v", err)
	}
	if len(phases) != 4 {
		t.Errorf("web resolved phases: got %d, want 4", len(phases))
	}
}

// --- ResolveTarget ---

func aptPhase() PhaseConfig {
	return PhaseConfig{Type: "apt", Apt: &AptConfig{Packages: []string{"curl"}}}
}

func TestResolveTarget_BackwardCompat_FlatPhases(t *testing.T) {
	cfg := &SprootConfig{Phases: []PhaseConfig{aptPhase()}}
	phases, err := cfg.ResolveTarget("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 1 || phases[0].Type != "apt" {
		t.Errorf("got %+v, want [apt]", phases)
	}
}

func TestResolveTarget_SingleTarget(t *testing.T) {
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"web": {Phases: []PhaseConfig{aptPhase()}},
		},
	}
	phases, err := cfg.ResolveTarget("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 1 || phases[0].Type != "apt" {
		t.Errorf("got %+v", phases)
	}
}

func TestResolveTarget_Extends(t *testing.T) {
	base := PhaseConfig{Type: "apt", Apt: &AptConfig{Packages: []string{"base-pkg"}}}
	extra := PhaseConfig{Type: "cmd", Cmd: &CmdConfig{Run: "extra"}}
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"base": {Phases: []PhaseConfig{base}},
			"web":  {Extends: "base", Phases: []PhaseConfig{extra}},
		},
	}
	phases, err := cfg.ResolveTarget("web")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 2 {
		t.Fatalf("got %d phases, want 2", len(phases))
	}
	if phases[0].Type != "apt" {
		t.Errorf("phases[0].type: got %q, want apt", phases[0].Type)
	}
	if phases[1].Type != "cmd" {
		t.Errorf("phases[1].type: got %q, want cmd", phases[1].Type)
	}
}

func TestResolveTarget_DefaultFallback(t *testing.T) {
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"default": {Phases: []PhaseConfig{aptPhase()}},
		},
	}
	phases, err := cfg.ResolveTarget("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(phases) != 1 {
		t.Errorf("got %d phases, want 1", len(phases))
	}
}

func TestResolveTarget_EmptyNameNoDefault(t *testing.T) {
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"web": {Phases: []PhaseConfig{aptPhase()}},
		},
	}
	_, err := cfg.ResolveTarget("")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "--target") {
		t.Errorf("error %q should direct the user to pass --target", err.Error())
	}
}

func TestResolveTarget_MissingTarget(t *testing.T) {
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"web": {Phases: []PhaseConfig{aptPhase()}},
		},
	}
	_, err := cfg.ResolveTarget("ml")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `"ml" not found`) {
		t.Errorf("error %q does not contain expected substring", err.Error())
	}
}

func TestResolveTarget_CycleDetected(t *testing.T) {
	cfg := &SprootConfig{
		Targets: map[string]*TargetConfig{
			"a": {Extends: "b", Phases: []PhaseConfig{aptPhase()}},
			"b": {Extends: "a", Phases: []PhaseConfig{aptPhase()}},
		},
	}
	_, err := cfg.ResolveTarget("a")
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "cycle") {
		t.Errorf("error %q does not mention cycle", err.Error())
	}
}

// --- ValidateSprootConfig targets ---

func TestValidateSprootConfig_TargetsValid(t *testing.T) {
	cfg := &SprootConfig{
		SchemaVersion: 1,
		Identity: Identity{
			GitUserName:      "Test",
			GitUserEmail:     "t@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "tester",
		},
		Targets: map[string]*TargetConfig{
			"base": {Phases: []PhaseConfig{aptPhase()}},
			"web":  {Extends: "base", Phases: []PhaseConfig{{Type: "cmd", Cmd: &CmdConfig{Run: "echo web"}}}},
		},
	}
	if err := ValidateSprootConfig(cfg); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateSprootConfig_PhasesPlusTargetsError(t *testing.T) {
	cfg := &SprootConfig{
		SchemaVersion: 1,
		Identity: Identity{
			GitUserName:      "Test",
			GitUserEmail:     "t@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "tester",
		},
		Phases:  []PhaseConfig{aptPhase()},
		Targets: map[string]*TargetConfig{"web": {Phases: []PhaseConfig{aptPhase()}}},
	}
	err := ValidateSprootConfig(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "phases and targets cannot both be set") {
		t.Errorf("error %q missing expected message", err.Error())
	}
}

func TestValidateSprootConfig_TargetDanglingExtends(t *testing.T) {
	cfg := &SprootConfig{
		SchemaVersion: 1,
		Identity: Identity{
			GitUserName:      "Test",
			GitUserEmail:     "t@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "tester",
		},
		Targets: map[string]*TargetConfig{
			"web": {Extends: "nonexistent", Phases: []PhaseConfig{aptPhase()}},
		},
	}
	err := ValidateSprootConfig(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `extends "nonexistent" not found`) {
		t.Errorf("error %q missing expected message", err.Error())
	}
}

// --- ValidateHostConfig with config_source ---

func TestValidateHostConfig_LocalSource(t *testing.T) {
	cfg := &HostConfig{
		SprootConfigSource:    "local",
		SprootConfigLocalPath: "~/my-config",
		TokenEnv:              "SPRITES_TOKEN",
	}
	if err := ValidateHostConfig(cfg); err != nil {
		t.Errorf("expected no error for local source: %v", err)
	}
}

func TestValidateHostConfig_LocalSourceMissingPath(t *testing.T) {
	cfg := &HostConfig{
		SprootConfigSource: "local",
		TokenEnv:           "SPRITES_TOKEN",
	}
	err := ValidateHostConfig(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "sproot_config_local_path is required") {
		t.Errorf("error %q missing expected message", err.Error())
	}
}

func TestValidateHostConfig_InvalidSource(t *testing.T) {
	cfg := &HostConfig{
		SprootConfigSource: "ftp",
		TokenEnv:           "SPRITES_TOKEN",
	}
	err := ValidateHostConfig(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), `"ftp" is not valid`) {
		t.Errorf("error %q missing expected message", err.Error())
	}
}

// writeFile is a test helper that writes content to a file.
func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o644)
}
