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
// It verifies that all 17 module types are registered and parse without panic.
func TestDryRunAllModules(t *testing.T) {
	// Provide files that file_template and rc_block need from the config repo.
	repoDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(repoDir, "file.txt"), []byte("hello\n"), 0o644)
	_ = os.WriteFile(filepath.Join(repoDir, "rc.sh"), []byte("export X=1\n"), 0o644)

	home := t.TempDir()
	t.Setenv("HOME", home)

	cfgs := []config.PhaseConfig{
		{Type: "apt", Apt: &config.AptConfig{Packages: []string{"bash"}}},
		{Type: "uv_tool", UVTool: &config.UVToolConfig{Tools: []config.UVTool{{Name: "ruff"}}}},
		{Type: "go_install", GoInstall: &config.GoInstallConfig{
			Tools: []config.GoTool{{Pkg: "golang.org/x/tools/cmd/goimports", Version: "latest"}},
		}},
		{Type: "cargo_install", CargoInstall: &config.CargoInstallConfig{
			Tools: []config.CargoTool{{Name: "ripgrep"}},
		}},
		{Type: "binary_release", BinaryRelease: &config.BinaryReleaseConfig{
			Name: "cosign", Repo: "sigstore/cosign", Asset: "cosign_{version}_{arch}.deb", Install: "dpkg",
		}},
		{Type: "corepack", Corepack: &config.CorepackConfig{Managers: []string{"pnpm", "yarn"}}},
		{Type: "rust_components", RustComponents: &config.RustComponentsConfig{Components: []string{"clippy", "rustfmt"}}},
		{Type: "docker", Docker: &config.DockerConfig{}},
		{Type: "sprite_service", SpriteService: &config.SpriteServiceConfig{
			Service: "dockerd", Cmd: "/usr/bin/dockerd",
		}},
		{Type: "git_identity", GitIdentity: &config.GitIdentityConfig{}},
		{Type: "ssh_setup", SSHSetup: &config.SSHSetupConfig{}},
		{Type: "gh_token", GHToken: &config.GHTokenConfig{}},
		{Type: "file_template", FileTemplate: &config.FileTemplateConfig{
			Src: "file.txt", Dest: filepath.Join(home, "out.txt"),
		}},
		{Type: "rc_block", RCBlock: &config.RCBlockConfig{Src: "rc.sh"}},
		{Type: "repo_clone", RepoClone: &config.RepoCloneConfig{
			BaseDir: filepath.Join(home, "repos"),
			Repos:   []string{"owner/repo"},
		}},
		{Type: "claude_settings", ClaudeSettings: &config.ClaudeSettingsConfig{
			Settings: map[string]any{"theme": "dark"},
		}},
		{Type: "cmd", Cmd: &config.CmdConfig{Run: "true"}},
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
