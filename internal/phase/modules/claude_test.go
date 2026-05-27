package modules

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

// claudeCtx builds a context with an explicit config-repo path (for claude_md src).
func claudeCtx(repoDir string) *phase.Context {
	return &phase.Context{
		ConfigRepoPath: repoDir,
		Log:            log.New(io.Discard),
		Identity: config.Identity{
			GitUserName:      "Test User",
			GitUserEmail:     "test@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "testuser",
		},
	}
}

func TestClaude_SettingsShouldRunWhenFileMissing(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{"theme": "dark"}}}
	should, err := p.ShouldRun(claudeCtx(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when settings.json missing")
	}
}

func TestClaude_SettingsRunAndVerify(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{"theme": "dark"}}}

	if err := p.Run(claudeCtx(t.TempDir())); err != nil {
		t.Fatalf("Run: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(home, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("read settings: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["theme"] != "dark" {
		t.Errorf("theme: got %v, want dark", got["theme"])
	}
	if err := p.Verify(claudeCtx(t.TempDir())); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestClaude_SettingsShouldRunFalseAfterRun(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{"theme": "dark"}}}
	if err := p.Run(claudeCtx(t.TempDir())); err != nil {
		t.Fatal(err)
	}
	should, err := p.ShouldRun(claudeCtx(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after settings written")
	}
}

func TestClaude_SettingsMergesIntoExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	data, _ := json.Marshal(map[string]any{"other": "value"})
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{"theme": "dark"}}}
	if err := p.Run(claudeCtx(t.TempDir())); err != nil {
		t.Fatal(err)
	}
	result, _ := readClaudeSettings()
	if result["other"] != "value" {
		t.Error("existing key 'other' was lost after merge")
	}
	if result["theme"] != "dark" {
		t.Error("new key 'theme' not set after merge")
	}
}

// TestClaude_NestedSettingsIdempotentWithExistingKeys guards against an exact
// top-level comparison of a deep-merged setting: when settings.json already has
// extra keys under a nested object, deepMerge preserves them, so the stored
// value is a superset of the configured value. ShouldRun and Verify must treat
// the configured nested map as a subset (present), not require exact equality,
// or the phase re-runs forever and Verify always fails.
func TestClaude_NestedSettingsIdempotentWithExistingKeys(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	claudeDir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	existing, _ := json.Marshal(map[string]any{
		"permissions": map[string]any{"defaultMode": "acceptEdits"},
	})
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), existing, 0o644)

	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{
		"permissions": map[string]any{"allow": []any{"Bash"}},
	}}}
	if err := p.Run(claudeCtx(t.TempDir())); err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Pre-existing nested key must be preserved by the deep merge.
	result, _ := readClaudeSettings()
	perms, _ := result["permissions"].(map[string]any)
	if perms["defaultMode"] != "acceptEdits" {
		t.Errorf("existing nested key lost after merge: %v", result["permissions"])
	}

	should, err := p.ShouldRun(claudeCtx(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after nested settings merged with pre-existing keys")
	}
	if err := p.Verify(claudeCtx(t.TempDir())); err != nil {
		t.Errorf("Verify should pass for a merged nested-settings superset: %v", err)
	}
}

func TestClaude_UpgradeAlwaysShouldRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Pre-write matching settings so the settings check alone would say "no work".
	p := &claudePhase{cfg: &config.ClaudeConfig{Settings: map[string]any{"theme": "dark"}}}
	if err := p.Run(claudeCtx(t.TempDir())); err != nil {
		t.Fatal(err)
	}
	// With upgrade set, ShouldRun must return true regardless.
	up := &claudePhase{cfg: &config.ClaudeConfig{Upgrade: true, Settings: map[string]any{"theme": "dark"}}}
	should, err := up.ShouldRun(claudeCtx(t.TempDir()))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when upgrade is set")
	}
}

func TestClaude_ClaudeMDWriteAndIdempotent(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("# Notes\nbe nice\n"), 0o644)

	p := &claudePhase{cfg: &config.ClaudeConfig{ClaudeMD: "CLAUDE.md"}}
	if err := p.Run(claudeCtx(repoDir)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	content, err := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if !strings.Contains(string(content), claudeMDBegin) || !strings.Contains(string(content), "be nice") {
		t.Errorf("managed block not written: %q", content)
	}
	if err := p.Verify(claudeCtx(repoDir)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	should, err := p.ShouldRun(claudeCtx(repoDir))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after CLAUDE.md block written")
	}
}

func TestClaude_ClaudeMDReplacesOnChange(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := t.TempDir()
	mdPath := filepath.Join(repoDir, "CLAUDE.md")

	_ = os.WriteFile(mdPath, []byte("first\n"), 0o644)
	p := &claudePhase{cfg: &config.ClaudeConfig{ClaudeMD: "CLAUDE.md"}}
	if err := p.Run(claudeCtx(repoDir)); err != nil {
		t.Fatal(err)
	}
	_ = os.WriteFile(mdPath, []byte("second\n"), 0o644)
	if err := p.Run(claudeCtx(repoDir)); err != nil {
		t.Fatal(err)
	}
	content, _ := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if strings.Count(string(content), claudeMDBegin) != 1 {
		t.Errorf("expected exactly one managed block, got: %q", content)
	}
	if strings.Contains(string(content), "first") || !strings.Contains(string(content), "second") {
		t.Errorf("block not replaced with new content: %q", content)
	}
}

func TestClaude_ClaudeMDTemplate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("user is {{.GitUserName}}\n"), 0o644)

	p := &claudePhase{cfg: &config.ClaudeConfig{ClaudeMD: "CLAUDE.md", Template: true}}
	if err := p.Run(claudeCtx(repoDir)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	content, _ := os.ReadFile(filepath.Join(home, ".claude", "CLAUDE.md"))
	if !strings.Contains(string(content), "user is Test User") {
		t.Errorf("template not rendered: %q", content)
	}
}

func TestClaude_CombinedSettingsAndClaudeMD(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	repoDir := t.TempDir()
	_ = os.WriteFile(filepath.Join(repoDir, "CLAUDE.md"), []byte("notes\n"), 0o644)

	p := &claudePhase{cfg: &config.ClaudeConfig{
		Settings: map[string]any{"theme": "dark"},
		ClaudeMD: "CLAUDE.md",
	}}
	if err := p.Run(claudeCtx(repoDir)); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); err != nil {
		t.Errorf("settings.json not written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "CLAUDE.md")); err != nil {
		t.Errorf("CLAUDE.md not written: %v", err)
	}
	if err := p.Verify(claudeCtx(repoDir)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}
