package modules

// claude configures Claude Code in the sprite. All three capabilities are optional:
//   - settings: deep-merged into ~/.claude/settings.json (existing keys preserved)
//   - upgrade:  runs `claude upgrade`
//   - claude_md: inserts a managed block into ~/.claude/CLAUDE.md from a config-repo file
//
//	- type: claude
//	  upgrade: true
//	  settings:
//	    theme: dark
//	  claude_md: files/CLAUDE.md
//	  template: true   # render claude_md as a Go template against the identity

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template" // nosemgrep

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const (
	claudeMDBegin = "<!-- BEGIN SPROOT MANAGED BLOCK -->"
	claudeMDEnd   = "<!-- END SPROOT MANAGED BLOCK -->"
)

func init() {
	phase.Register("claude", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Claude == nil {
			return nil, fmt.Errorf("claude: config is nil")
		}
		return &claudePhase{cfg: cfg.Claude}, nil
	})
}

type claudePhase struct {
	cfg *config.ClaudeConfig
}

func (p *claudePhase) Type() string { return "claude" }
func (p *claudePhase) Name() string { return "claude" }

func (p *claudePhase) ShouldRun(ctx *phase.Context) (bool, error) {
	// upgrade has no cheap idempotency check, so when requested the phase runs.
	if p.cfg.Upgrade {
		return true, nil
	}
	if len(p.cfg.Settings) > 0 {
		if p.settingsNeedWork() {
			return true, nil
		}
	}
	if p.cfg.ClaudeMD != "" {
		need, err := p.claudeMDNeedsWork(ctx)
		if err != nil || need {
			return true, err
		}
	}
	return false, nil
}

func (p *claudePhase) Run(ctx *phase.Context) error {
	if len(p.cfg.Settings) > 0 {
		if err := p.applySettings(ctx); err != nil {
			return err
		}
	}
	if p.cfg.ClaudeMD != "" {
		if err := p.applyClaudeMD(ctx); err != nil {
			return err
		}
	}
	if p.cfg.Upgrade {
		if err := runCmd(ctx.Log, "claude", "upgrade"); err != nil {
			return fmt.Errorf("claude: upgrade: %w", err)
		}
	}
	return nil
}

func (p *claudePhase) Verify(ctx *phase.Context) error {
	if len(p.cfg.Settings) > 0 {
		existing, err := readClaudeSettings()
		if err != nil {
			return fmt.Errorf("claude: read settings: %w", err)
		}
		for k, want := range p.cfg.Settings {
			got, ok := existing[k]
			if !ok || !settingsSatisfied(got, want) {
				return fmt.Errorf("claude: settings key %q not set correctly", k)
			}
		}
	}
	if p.cfg.ClaudeMD != "" {
		src, err := p.renderClaudeMD(ctx)
		if err != nil {
			return err
		}
		path, err := claudeMDPath()
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("claude: cannot read %s: %w", path, err)
		}
		if extractManagedBlockHash(string(content), claudeMDBegin, claudeMDEnd) != blockHash(ensureTrailingNewline(string(src))) {
			return fmt.Errorf("claude: CLAUDE.md block missing or stale")
		}
	}
	return nil
}

// --- settings ---

func (p *claudePhase) settingsNeedWork() bool {
	existing, err := readClaudeSettings()
	if err != nil {
		return true
	}
	for k, want := range p.cfg.Settings {
		got, ok := existing[k]
		if !ok || !settingsSatisfied(got, want) {
			return true
		}
	}
	return false
}

// settingsSatisfied reports whether want is already reflected in got, mirroring
// deepMerge: nested maps are satisfied key-by-key (got may carry extra keys that
// the merge preserves), while non-map values must match exactly. An exact
// top-level comparison would wrongly flag a merged superset as out of date and
// re-run (and fail Verify) on every invocation.
func settingsSatisfied(got, want any) bool {
	wantMap, ok := want.(map[string]any)
	if !ok {
		// Compare via JSON to handle int/float/bool mismatches from YAML decode.
		return jsonStr(got) == jsonStr(want)
	}
	gotMap, ok := got.(map[string]any)
	if !ok {
		return false
	}
	for k, wv := range wantMap {
		gv, ok := gotMap[k]
		if !ok || !settingsSatisfied(gv, wv) {
			return false
		}
	}
	return true
}

func (p *claudePhase) applySettings(ctx *phase.Context) error {
	existing, _ := readClaudeSettings()
	if existing == nil {
		existing = make(map[string]any)
	}
	deepMerge(existing, p.cfg.Settings)

	path, err := claudeSettingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("claude: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(existing, "", "  ")
	if err != nil {
		return fmt.Errorf("claude: marshal settings: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o644); err != nil {
		return fmt.Errorf("claude: write settings: %w", err)
	}
	ctx.Log.Infof("wrote %s", path)
	return nil
}

func readClaudeSettings() (map[string]any, error) {
	path, err := claudeSettingsPath()
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

func claudeSettingsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

// --- CLAUDE.md managed block ---

func (p *claudePhase) claudeMDNeedsWork(ctx *phase.Context) (bool, error) {
	src, err := p.renderClaudeMD(ctx)
	if err != nil {
		return false, err
	}
	path, err := claudeMDPath()
	if err != nil {
		return false, err
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return true, nil
	}
	return extractManagedBlockHash(string(content), claudeMDBegin, claudeMDEnd) != blockHash(ensureTrailingNewline(string(src))), nil
}

func (p *claudePhase) applyClaudeMD(ctx *phase.Context) error {
	src, err := p.renderClaudeMD(ctx)
	if err != nil {
		return err
	}
	path, err := claudeMDPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("claude: mkdir: %w", err)
	}
	if err := applyManagedBlock(path, string(src), claudeMDBegin, claudeMDEnd); err != nil {
		return fmt.Errorf("claude: write CLAUDE.md: %w", err)
	}
	ctx.Log.Infof("wrote block to %s", path)
	return nil
}

func (p *claudePhase) renderClaudeMD(ctx *phase.Context) ([]byte, error) {
	raw, err := os.ReadFile(filepath.Join(ctx.ConfigRepoPath, p.cfg.ClaudeMD))
	if err != nil {
		return nil, fmt.Errorf("claude: read claude_md %q: %w", p.cfg.ClaudeMD, err)
	}
	if !p.cfg.Template {
		return raw, nil
	}
	tmpl, err := template.New(filepath.Base(p.cfg.ClaudeMD)).Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("claude: parse claude_md template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.Identity); err != nil {
		return nil, fmt.Errorf("claude: render claude_md template: %w", err)
	}
	return buf.Bytes(), nil
}

func claudeMDPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "CLAUDE.md"), nil
}

// ensureTrailingNewline mirrors applyManagedBlock's normalization so hash
// comparisons in ShouldRun/Verify match what Run actually wrote.
func ensureTrailingNewline(s string) string {
	if len(s) == 0 || s[len(s)-1] != '\n' {
		return s + "\n"
	}
	return s
}
