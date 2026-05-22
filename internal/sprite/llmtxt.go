package sprite

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

// defaultSpriteContextDir is the in-sprite directory that holds platform and
// tool context files. The sprite platform ships its own llm.txt and
// docs/agent-context.md here; sproot must not clobber those.
const defaultSpriteContextDir = "/.sprite"

// sprootSummaryRel is the path (relative to the context dir) of sproot's own
// setup summary. It lives alongside the platform docs without overwriting them.
const sprootSummaryRel = "docs/sproot-setup.md"

// pointerBegin/pointerEnd delimit the managed block sproot appends to the
// platform's llm.txt so agents reading the canonical entrypoint discover the
// sproot summary. The block is replaced (not duplicated) on re-run.
const (
	pointerBegin = "<!-- BEGIN SPROOT -->"
	pointerEnd   = "<!-- END SPROOT -->"
)

// moduleDescriptions maps a phase Type to a one-line description of what it
// provisions. Used to render the post-setup summary. Unknown types fall back
// to the phase name.
var moduleDescriptions = map[string]string{
	"apt":             "Installed system packages via apt.",
	"uv_tool":         "Installed Python CLI tools via uv.",
	"go_install":      "Installed Go binaries via go install.",
	"cargo_install":   "Installed Rust binaries via cargo install.",
	"binary_release":  "Installed binaries from GitHub release assets.",
	"corepack":        "Enabled corepack package managers (yarn, pnpm).",
	"rust_components": "Installed Rust toolchain components via rustup.",
	"docker":          "Installed and configured Docker.",
	"sprite_service":  "Registered a long-running sprite service.",
	"git_identity":    "Configured git user identity.",
	"ssh_setup":       "Generated an SSH key and registered it with GitHub.",
	"gh_token":        "Exported GH_TOKEN and authenticated gh.",
	"file_template":   "Rendered files into place from the config repo.",
	"rc_block":        "Added a managed block to shell rc files.",
	"repo_clone":      "Cloned repositories.",
	"claude":          "Configured Claude Code (settings, upgrade, CLAUDE.md).",
	"npm":             "Installed Node.js dependencies via npm install.",
	"cmd":             "Ran custom setup commands.",
}

// renderLLMContext builds a markdown summary of a completed setup run. It lists
// the modules that did work (with descriptions) and a one-line tally. The output
// is deterministic for a given state except for the trailing timestamp.
func renderLLMContext(state *phase.State) string {
	var b strings.Builder
	b.WriteString("# sproot setup summary\n\n")
	b.WriteString("This sprite was provisioned by sproot. The following describes what setup installed.\n\n")
	fmt.Fprintf(&b, "Generated at %s.\n\n", state.UpdatedAt.Format("2006-01-02 15:04:05 UTC"))

	var didWork, skipped, failed int
	var worked []phase.PhaseRecord
	for _, rec := range state.Phases {
		switch {
		case rec.Error != "" || rec.VerifyError != "":
			failed++
		case rec.Skipped:
			skipped++
		default:
			didWork++
			if rec.DidWork {
				worked = append(worked, rec)
			}
		}
	}

	b.WriteString("## What was provisioned\n\n")
	if len(worked) == 0 {
		b.WriteString("Nothing changed on this run (all phases were already satisfied).\n\n")
	} else {
		for _, rec := range worked {
			desc := moduleDescriptions[rec.Type]
			if desc == "" {
				desc = rec.Name
			}
			fmt.Fprintf(&b, "- **%s**: %s\n", rec.Name, desc)
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "Summary: %d phase(s) did work, %d already satisfied, %d failed.\n",
		didWork, skipped, failed)
	return b.String()
}

// writeLLMContext writes sproot's setup summary to baseDir/docs/sproot-setup.md
// and appends an idempotent pointer block to baseDir/llm.txt so agents reading
// the platform's entrypoint find it. It never overwrites platform-owned files.
func writeLLMContext(l *log.Logger, state *phase.State, baseDir string) error {
	content := renderLLMContext(state)

	summaryPath := filepath.Join(baseDir, sprootSummaryRel)
	if err := os.MkdirAll(filepath.Dir(summaryPath), 0o755); err != nil {
		return fmt.Errorf("create context dir: %w", err)
	}
	if err := os.WriteFile(summaryPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", summaryPath, err)
	}

	if err := appendPointerBlock(filepath.Join(baseDir, "llm.txt"), summaryPath); err != nil {
		return fmt.Errorf("update llm.txt pointer: %w", err)
	}

	l.Debugf("wrote sproot setup summary to %s", summaryPath)
	return nil
}

var pointerRe = regexp.MustCompile(`(?s)\n?` + regexp.QuoteMeta(pointerBegin) + `.*?` + regexp.QuoteMeta(pointerEnd) + `\n?`)

// appendPointerBlock strips any existing sproot pointer block from the file at
// path, then appends a fresh one referencing summaryPath. The file (the
// platform's llm.txt) may not exist yet; in that case the block becomes its
// whole content.
func appendPointerBlock(path, summaryPath string) error {
	existing, _ := os.ReadFile(path)
	stripped := strings.TrimRight(pointerRe.ReplaceAllString(string(existing), ""), "\n")

	block := fmt.Sprintf("%s\n## sproot\n\nThis sprite was provisioned by sproot. See %s for a summary of what setup installed.\n%s\n",
		pointerBegin, summaryPath, pointerEnd)

	var out string
	if stripped == "" {
		out = block
	} else {
		out = stripped + "\n\n" + block
	}
	return os.WriteFile(path, []byte(out), 0o644)
}
