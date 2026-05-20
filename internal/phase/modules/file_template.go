package modules

// file_template copies a file from the config repo to a destination path.
// When template: true is set, the file is rendered as a Go template against
// the identity struct. Tilde in dest is expanded to the home directory.
//
//	- type: file_template
//	  src: files/statusline.py
//	  dest: ~/.claude/statusline.py
//	  mode: "0755"      # optional; octal string
//	  template: false   # set true to render Go template variables

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"text/template"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("file_template", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.FileTemplate == nil {
			return nil, fmt.Errorf("file_template: config is nil")
		}
		return &fileTemplatePhase{cfg: cfg.FileTemplate}, nil
	})
}

type fileTemplatePhase struct {
	cfg *config.FileTemplateConfig
}

func (p *fileTemplatePhase) Type() string { return "file_template" }
func (p *fileTemplatePhase) Name() string { return fmt.Sprintf("file_template(%s)", p.cfg.Dest) }

func (p *fileTemplatePhase) ShouldRun(ctx *phase.Context) (bool, error) {
	want, err := p.render(ctx)
	if err != nil {
		return true, nil
	}
	dest, err := expandTilde(p.cfg.Dest)
	if err != nil {
		return true, nil
	}
	existing, err := os.ReadFile(dest)
	if err != nil {
		return true, nil
	}
	return !bytes.Equal(existing, want), nil
}

func (p *fileTemplatePhase) Run(ctx *phase.Context) error {
	rendered, err := p.render(ctx)
	if err != nil {
		return err
	}
	dest, err := expandTilde(p.cfg.Dest)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return fmt.Errorf("file_template: mkdir: %w", err)
	}
	perm := os.FileMode(0o644)
	if p.cfg.Mode != "" {
		v, err := strconv.ParseUint(p.cfg.Mode, 8, 32)
		if err != nil {
			return fmt.Errorf("file_template: invalid mode %q: %w", p.cfg.Mode, err)
		}
		perm = os.FileMode(v)
	}
	if err := os.WriteFile(dest, rendered, perm); err != nil {
		return fmt.Errorf("file_template: write %s: %w", dest, err)
	}
	ctx.Log.Infof("wrote %s", dest)
	return nil
}

func (p *fileTemplatePhase) Verify(ctx *phase.Context) error {
	want, err := p.render(ctx)
	if err != nil {
		return err
	}
	dest, err := expandTilde(p.cfg.Dest)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(dest)
	if err != nil {
		return fmt.Errorf("file_template: %s missing after write", dest)
	}
	if !bytes.Equal(existing, want) {
		return fmt.Errorf("file_template: %s content does not match source", dest)
	}
	return nil
}

func (p *fileTemplatePhase) render(ctx *phase.Context) ([]byte, error) {
	srcPath := filepath.Join(ctx.ConfigRepoPath, p.cfg.Src)
	raw, err := os.ReadFile(srcPath)
	if err != nil {
		return nil, fmt.Errorf("file_template: read src %q: %w", p.cfg.Src, err)
	}
	if !p.cfg.Template {
		return raw, nil
	}
	tmpl, err := template.New("").Parse(string(raw))
	if err != nil {
		return nil, fmt.Errorf("file_template: parse template %q: %w", p.cfg.Src, err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx.Identity); err != nil {
		return nil, fmt.Errorf("file_template: execute template: %w", err)
	}
	return buf.Bytes(), nil
}

// expandTilde replaces a leading ~ with the user home directory.
func expandTilde(path string) (string, error) {
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if len(path) == 1 {
		return home, nil
	}
	return filepath.Join(home, path[2:]), nil
}
