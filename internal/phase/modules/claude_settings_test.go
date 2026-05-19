package modules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestClaudeSettings_ShouldRunWhenFileMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := &claudeSettingsPhase{cfg: &config.ClaudeSettingsConfig{
		Settings: map[string]any{"theme": "dark"},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when settings.json missing")
	}
}

func TestClaudeSettings_RunAndVerify(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := &claudeSettingsPhase{cfg: &config.ClaudeSettingsConfig{
		Settings: map[string]any{"theme": "dark"},
	}}

	if err := p.Run(testCtx(t)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	path := filepath.Join(home, ".claude", "settings.json")
	data, err := os.ReadFile(path)
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

	if err := p.Verify(testCtx(t)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestClaudeSettings_ShouldRunFalseAfterRun(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	p := &claudeSettingsPhase{cfg: &config.ClaudeSettingsConfig{
		Settings: map[string]any{"theme": "dark"},
	}}
	if err := p.Run(testCtx(t)); err != nil {
		t.Fatal(err)
	}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false after settings written")
	}
}

func TestClaudeSettings_MergesIntoExisting(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Write pre-existing settings file.
	claudeDir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(claudeDir, 0o755)
	existing := map[string]any{"other": "value"}
	data, _ := json.Marshal(existing)
	_ = os.WriteFile(filepath.Join(claudeDir, "settings.json"), data, 0o644)

	p := &claudeSettingsPhase{cfg: &config.ClaudeSettingsConfig{
		Settings: map[string]any{"theme": "dark"},
	}}
	if err := p.Run(testCtx(t)); err != nil {
		t.Fatal(err)
	}

	result, _ := p.readSettings()
	if result["other"] != "value" {
		t.Error("existing key 'other' was lost after merge")
	}
	if result["theme"] != "dark" {
		t.Error("new key 'theme' not set after merge")
	}
}

func TestDeepMerge(t *testing.T) {
	dst := map[string]any{
		"a": "original",
		"nested": map[string]any{
			"x": 1,
		},
	}
	src := map[string]any{
		"b": "new",
		"nested": map[string]any{
			"y": 2,
		},
	}
	deepMerge(dst, src)

	if dst["a"] != "original" {
		t.Error("existing key 'a' was overwritten")
	}
	if dst["b"] != "new" {
		t.Error("new key 'b' missing after merge")
	}
	nested, _ := dst["nested"].(map[string]any)
	if nested["x"] != 1 {
		t.Error("nested key 'x' lost after merge")
	}
	if nested["y"] != 2 {
		t.Error("nested key 'y' missing after merge")
	}
}
