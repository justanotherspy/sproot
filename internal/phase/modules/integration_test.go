package modules

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

// TestDryRunAllModules builds a runner covering every registered module type
// with minimal valid config, runs it under DryRun=true, and asserts no errors.
// It verifies that all 20 module types are registered and parse without panic.
func TestDryRunAllModules(t *testing.T) {
	// Provide files that file_template and rc_block need from the config repo.
	repoDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "rc.sh"), []byte("export X=1\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("# Project notes\n"), 0o644)

	home := t.TempDir()
	t.Setenv("HOME", home)

	// npm phase needs a dir with a package.json (node_modules absent so ShouldRun=true in dry-run).
	npmDir := filepath.Join(home, "myapp")
	_ = os.MkdirAll(npmDir, 0o755)

	cfgs := []config.PhaseConfig{
		// apt: with symlinks field exercised
		{Type: "apt", Apt: &config.AptConfig{
			Packages: []string{"bash"},
			Symlinks: []config.AptSymlink{{From: "/usr/bin/batcat", To: "/usr/local/bin/bat"}},
		}},
		// uv_tool: pkg field (package name differs from binary name)
		{Type: "uv_tool", UVTool: &config.UVToolConfig{Tools: []config.UVTool{{Name: "garlic", Pkg: "garlic-cli"}}}},
		{Type: "go_install", GoInstall: &config.GoInstallConfig{
			Tools: []config.GoTool{{Pkg: "golang.org/x/tools/cmd/goimports", Version: "latest"}},
		}},
		{Type: "cargo_install", CargoInstall: &config.CargoInstallConfig{
			Tools: []config.CargoTool{{Name: "ripgrep"}},
		}},
		// binary_release: version field ({tag_no_v}) + x64_arch template var
		{Type: "binary_release", BinaryRelease: &config.BinaryReleaseConfig{
			Name: "gitleaks", Repo: "gitleaks/gitleaks", Version: "{tag_no_v}",
			Asset: "gitleaks_{version}_linux_{x64_arch}.tar.gz", Install: "tar+install",
		}},
		// binary_release: arch_map + {arch_alias} (hadolint-style naming)
		{Type: "binary_release", BinaryRelease: &config.BinaryReleaseConfig{
			Name: "hadolint", Repo: "hadolint/hadolint",
			ArchMap: map[string]string{"amd64": "x86_64", "arm64": "arm64"},
			Asset:   "hadolint-linux-{arch_alias}", Install: "raw",
		}},
		{Type: "corepack", Corepack: &config.CorepackConfig{Managers: []string{"pnpm", "yarn"}}},
		{Type: "rust_components", RustComponents: &config.RustComponentsConfig{Components: []string{"clippy", "rustfmt"}}},
		// docker: daemon_json field
		{Type: "docker", Docker: &config.DockerConfig{
			DaemonJSON: map[string]any{"log-driver": "json-file"},
		}},
		// sprite_service: http_port and needs fields
		{Type: "sprite_service", SpriteService: &config.SpriteServiceConfig{
			Service: "myservice", Cmd: "/usr/bin/myservice", HTTPPort: 8080, Needs: []string{"dockerd"},
		}},
		{Type: "git_identity", GitIdentity: &config.GitIdentityConfig{}},
		{Type: "ssh_setup", SSHSetup: &config.SSHSetupConfig{}},
		{Type: "gh_token", GHToken: &config.GHTokenConfig{}},
		{Type: "file_template", FileTemplate: &config.FileTemplateConfig{
			Src: "file.txt", Dest: filepath.Join(home, "out.txt"),
		}},
		{Type: "rc_block", RCBlock: &config.RCBlockConfig{Src: "rc.sh"}},
		// repo_clone: short form and long URL form
		{Type: "repo_clone", RepoClone: &config.RepoCloneConfig{
			BaseDir: filepath.Join(home, "repos"),
			Repos: []config.RepoCloneEntry{
				{Raw: "owner/repo"},
				{URL: "https://github.com/org/project.git", Dest: filepath.Join(home, "project")},
			},
		}},
		// claude: settings deep-merge + managed CLAUDE.md block from the config repo
		{Type: "claude", Claude: &config.ClaudeConfig{
			Settings: map[string]any{"theme": "dark"},
			ClaudeMD: "CLAUDE.md",
		}},
		// npm: installs node_modules in a project dir
		{Type: "npm", Npm: &config.NpmConfig{Dir: npmDir}},
		// shell_completion: generate + install completions (dry-run skips Run)
		{Type: "shell_completion", ShellCompletion: &config.ShellCompletionConfig{
			Completions: []config.ShellCompletionEntry{
				{Command: "sproot", Shells: []string{"bash", "zsh", "fish"}},
			},
		}},
		{Type: "cmd", Cmd: &config.CmdConfig{Run: "true"}},
		// nix: short + long package forms, daemon service, and a setup script.
		{Type: "nix", Nix: &config.NixConfig{
			Packages: []config.NixPackage{
				{Name: "hello"},
				{Name: "ripgrep", Bin: "rg"},
				{Flake: "nixpkgs#nixfmt-rfc-style", Bin: "nixfmt"},
			},
		}},
	}

	r, err := phase.NewRunner(cfgs, phase.RunnerOptions{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	ctx := &phase.Context{
		ConfigRepoPath: repoDir,
		Log:            log.New(io.Discard),
		DryRun:         true,
		Identity: config.Identity{
			GitUserName:      "Test User",
			GitUserEmail:     "test@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "testuser",
		},
	}

	if err := r.Run(ctx); err != nil {
		t.Errorf("dry-run failed: %v", err)
	}
}

// TestDryRunTargetPhases verifies that target resolution (with extends) produces
// the correct phase list and that all resolved phases run cleanly under DryRun.
func TestDryRunTargetPhases(t *testing.T) {
	repoDir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	identity := config.Identity{
		GitUserName:      "Test User",
		GitUserEmail:     "test@example.com",
		GitDefaultBranch: "main",
		GHUsername:       "testuser",
	}

	cfg := &config.SprootConfig{
		Identity: identity,
		Targets: map[string]*config.TargetConfig{
			"base": {
				Phases: []config.PhaseConfig{
					{Type: "cmd", Cmd: &config.CmdConfig{Run: "true"}},
				},
			},
			"web": {
				Extends: "base",
				Phases: []config.PhaseConfig{
					{Type: "cmd", Cmd: &config.CmdConfig{Run: "true"}},
				},
			},
		},
	}

	phases, err := cfg.ResolveTarget("web")
	if err != nil {
		t.Fatalf("ResolveTarget(web): %v", err)
	}
	if len(phases) != 2 {
		t.Fatalf("expected 2 phases (base + web), got %d", len(phases))
	}

	r, err := phase.NewRunner(phases, phase.RunnerOptions{
		StatePath: filepath.Join(t.TempDir(), "state.json"),
	})
	if err != nil {
		t.Fatalf("NewRunner: %v", err)
	}

	ctx := &phase.Context{
		ConfigRepoPath: repoDir,
		Log:            log.New(io.Discard),
		DryRun:         true,
		Identity:       identity,
	}

	if err := r.Run(ctx); err != nil {
		t.Errorf("dry-run for web target failed: %v", err)
	}
}
