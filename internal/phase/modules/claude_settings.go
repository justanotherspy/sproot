package modules

// claude_settings deep-merges a JSON object into ~/.claude/settings.json.
// Existing keys not in the settings map are preserved.
//
//	- type: claude_settings
//	  settings:
//	    theme: dark
//	    autoUpdaterStatus: disabled

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("claude_settings", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.ClaudeSettings == nil {
			return nil, fmt.Errorf("claude_settings: config is nil")
		}
		return &claudeSettingsPhase{cfg: cfg.ClaudeSettings}, nil
	})
}

type claudeSettingsPhase struct {
	cfg *config.ClaudeSettingsConfig
}

func (p *claudeSettingsPhase) Type() string { return "claude_settings" }
func (p *claudeSettingsPhase) Name() string { return "claude_settings" }

func (p *claudeSettingsPhase) ShouldRun(_ *phase.Context) (bool, error) {
	existing, err := p.readSettings()
	if err != nil {
		return true, nil
	}
	for k, want := range p.cfg.Settings {
		got, ok := existing[k]
		if !ok {
			return true, nil
		}
		// Compare via JSON round-trip to handle int/float/bool mismatches from YAML decode.
		if jsonStr(got) != jsonStr(want) {
			return true, nil
		}
	}
	return false, nil
}

func (p *claudeSettingsPhase) Run(ctx *phase.Context) error {
	existing, _ := p.readSettings()
	if existing == nil {
		existing = make(map[string]any)
	}
	deepMerge(existing, p.cfg.Settings)

	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("claude_settings: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("claude_settings: marshal: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("claude_settings: write: %w", err)
	}
	ctx.Log.Infof("wrote %s", path)
	return nil
}

func (p *claudeSettingsPhase) Verify(_ *phase.Context) error {
	existing, err := p.readSettings()
	if err != nil {
		return fmt.Errorf("claude_settings: read settings: %w", err)
	}
	for k, want := range p.cfg.Settings {
		got, ok := existing[k]
		if !ok || jsonStr(got) != jsonStr(want) {
			return fmt.Errorf("claude_settings: key %q not set correctly", k)
		}
	}
	return nil
}

func (p *claudeSettingsPhase) readSettings() (map[string]any, error) {
	path, err := settingsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func settingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// deepMerge merges src into dst. Nested maps are merged recursively; other types are overwritten.
func deepMerge(dst, src map[string]any) {
	for k, sv := range src {
		if sm, ok := sv.(map[string]any); ok {
			if dm, ok := dst[k].(map[string]any); ok {
				deepMerge(dm, sm)
				continue
			}
		}
		dst[k] = sv
	}
}

// jsonStr serializes v to a JSON string for comparison purposes.
func jsonStr(v any) string {
	b, _ := json.Marshal(v)
	return string(b)
}
