package sprite

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

// defaultSpriteContextDir is the in-sprite directory where setup writes its
// post-run summary. Claude Code reads from here for instant environment context.
const defaultSpriteContextDir = "/.sprite"

// moduleDescriptions maps a phase Type to a one-line description of what it
// provisions. Used to render the post-setup summary. The verify phase and any
// unknown type fall back to the phase name.
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
	"claude_settings": "Installed Claude Code settings.",
	"npm":             "Installed Node.js dependencies via npm install.",
	"cmd":             "Ran custom setup commands.",
}

// renderLLMContext builds a markdown summary of a completed setup run. It lists
// the modules that did work (with descriptions) and a one-line tally. The output
// is deterministic for a given state except for the trailing timestamp.
func renderLLMContext(state *phase.State) string {
	var b strings.Builder
	b.WriteString("# Sprite environment\n\n")
	b.WriteString("This sprite was provisioned by sproot. The following describes what was set up.\n\n")
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

// writeLLMContext renders the run summary and writes it to baseDir/llm.txt and
// baseDir/docs/agent-context.md, creating directories as needed. Both files
// receive identical content.
func writeLLMContext(l *log.Logger, state *phase.State, baseDir string) error {
	content := renderLLMContext(state)

	docsDir := filepath.Join(baseDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		return fmt.Errorf("create context dir: %w", err)
	}

	llmPath := filepath.Join(baseDir, "llm.txt")
	if err := os.WriteFile(llmPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", llmPath, err)
	}
	agentPath := filepath.Join(docsDir, "agent-context.md")
	if err := os.WriteFile(agentPath, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", agentPath, err)
	}

	l.Debugf("wrote sprite context to %s and %s", llmPath, agentPath)
	return nil
}
