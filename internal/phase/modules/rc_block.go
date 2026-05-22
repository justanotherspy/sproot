package modules

// rc_block writes a sentinel-delimited block from a source file into ~/.bashrc and ~/.zshrc.
// Existing blocks with the same sentinel are replaced on re-run.
//
//	- type: rc_block
//	  src: files/rc_additions.sh

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const (
	rcBegin = "# BEGIN SPROOT MANAGED BLOCK"
	rcEnd   = "# END SPROOT MANAGED BLOCK"
)

func init() {
	phase.Register("rc_block", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.RCBlock == nil {
			return nil, fmt.Errorf("rc_block: config is nil")
		}
		return &rcBlockPhase{cfg: cfg.RCBlock}, nil
	})
}

type rcBlockPhase struct {
	cfg *config.RCBlockConfig
}

func (p *rcBlockPhase) Type() string { return "rc_block" }
func (p *rcBlockPhase) Name() string { return "rc_block" }

func (p *rcBlockPhase) ShouldRun(ctx *phase.Context) (bool, error) {
	src, err := p.readSrc(ctx)
	if err != nil {
		return true, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return true, nil
	}
	wantHash := blockHash(src)
	for _, rc := range []string{".bashrc", ".zshrc"} {
		existing, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			return true, nil
		}
		if extractManagedBlockHash(string(existing), rcBegin, rcEnd) != wantHash {
			return true, nil
		}
	}
	return false, nil
}

func (p *rcBlockPhase) Run(ctx *phase.Context) error {
	src, err := p.readSrc(ctx)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		if err := applyManagedBlock(path, src, rcBegin, rcEnd); err != nil {
			return fmt.Errorf("rc_block: %s: %w", rc, err)
		}
		ctx.Log.Infof("wrote block to %s", path)
	}
	return nil
}

func (p *rcBlockPhase) Verify(ctx *phase.Context) error {
	src, err := p.readSrc(ctx)
	if err != nil {
		return err
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	want := blockHash(src)
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("rc_block: cannot read %s: %w", rc, err)
		}
		if extractManagedBlockHash(string(content), rcBegin, rcEnd) != want {
			return fmt.Errorf("rc_block: block missing or stale in %s", rc)
		}
	}
	return nil
}

func (p *rcBlockPhase) readSrc(ctx *phase.Context) (string, error) {
	data, err := os.ReadFile(filepath.Join(ctx.ConfigRepoPath, p.cfg.Src))
	if err != nil {
		return "", fmt.Errorf("rc_block: read src %q: %w", p.cfg.Src, err)
	}
	return string(data), nil
}

// applyManagedBlock strips any existing begin/end-delimited block from the file,
// then appends a fresh one wrapping src. Shared by rc_block and claude.
func applyManagedBlock(path, src, begin, end string) error {
	if !strings.HasSuffix(src, "\n") {
		src += "\n"
	}
	existing, _ := os.ReadFile(path)
	stripped := stripManagedBlock(string(existing), begin, end)
	block := fmt.Sprintf("\n%s\n%s%s\n", begin, src, end)
	return os.WriteFile(path, []byte(stripped+block), 0o644)
}

// stripManagedBlock removes any existing begin/end-delimited block from content.
func stripManagedBlock(content, begin, end string) string {
	re := regexp.MustCompile(`(?s)\n?` + regexp.QuoteMeta(begin) + `\n.*?` + regexp.QuoteMeta(end) + `\n?`)
	return re.ReplaceAllString(content, "")
}

// blockHash returns a hex SHA-256 of the block body (for idempotency comparison).
func blockHash(src string) string {
	h := sha256.Sum256([]byte(src))
	return fmt.Sprintf("%x", h)
}

// extractManagedBlockHash reads the block body delimited by begin/end and returns its hash.
func extractManagedBlockHash(content, begin, end string) string {
	start := strings.Index(content, begin)
	endIdx := strings.Index(content, end)
	if start < 0 || endIdx < 0 || endIdx <= start {
		return ""
	}
	body := content[start+len(begin)+1 : endIdx]
	return blockHash(body)
}
