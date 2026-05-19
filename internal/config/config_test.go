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
	if len(cfg.Phases) != 15 {
		t.Fatalf("phases: got %d, want 15", len(cfg.Phases))
	}
	if cfg.Phases[0].Type != "apt" {
		t.Errorf("phases[0].type: got %q, want apt", cfg.Phases[0].Type)
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
	if rc.Repos[0] != "justanotherspy/garlic" {
		t.Errorf("repos[0]: got %q", rc.Repos[0])
	}
}

func TestPhaseConfig_CmdFields(t *testing.T) {
	cfg, err := LoadSprootConfig(testdataPath("sproot.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	p := cfg.Phases[14]
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
		{6, "corepack", func(p PhaseConfig) bool { return p.Corepack != nil }},
		{7, "rust_components", func(p PhaseConfig) bool { return p.RustComponents != nil }},
		{8, "docker", func(p PhaseConfig) bool { return p.Docker != nil }},
		{10, "git_identity", func(p PhaseConfig) bool { return p.GitIdentity != nil }},
		{11, "ssh_setup", func(p PhaseConfig) bool { return p.SSHSetup != nil }},
		{12, "gh_token", func(p PhaseConfig) bool { return p.GHToken != nil }},
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
	if cfg.ConfigRepo != "git@github.com:justanotherspy/sprite.git" {
		t.Errorf("config_repo: got %q", cfg.ConfigRepo)
	}
	if cfg.ConfigRef != "main" {
		t.Errorf("config_ref: got %q", cfg.ConfigRef)
	}
	if cfg.PrivateKey != "~/.sproot/private/id_ed25519" {
		t.Errorf("private_key: got %q", cfg.PrivateKey)
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
			"phases must not be empty",
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
				c.Phases = []PhaseConfig{{Type: "repo_clone", RepoClone: &RepoCloneConfig{Repos: []string{"a/b"}}}}
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
			"cmd_missing_run",
			func(c *SprootConfig) {
				c.Phases = []PhaseConfig{{Type: "cmd", Cmd: &CmdConfig{}}}
			},
			"phases[0] (cmd): run is required",
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
			ConfigRepo: "git@github.com:user/repo.git",
			ConfigRef:  "main",
			PrivateKey: "~/.sproot/private/id_ed25519",
		}
	}

	cases := []struct {
		name    string
		mutate  func(*HostConfig)
		wantErr string
	}{
		{"missing_config_repo", func(c *HostConfig) { c.ConfigRepo = "" }, "config_repo is required"},
		{"missing_config_ref", func(c *HostConfig) { c.ConfigRef = "" }, "config_ref is required"},
		{"missing_private_key", func(c *HostConfig) { c.PrivateKey = "" }, "private_key is required"},
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
	if !strings.HasSuffix(path, ".sproot/config") {
		t.Errorf("path %q does not end with .sproot/config", path)
	}
}

// writeFile is a test helper that writes content to a file.
func writeFile(t *testing.T, path, content string) error {
	t.Helper()
	return os.WriteFile(path, []byte(content), 0o644)
}
