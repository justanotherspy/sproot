package phase

import (
	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// Context holds the shared runtime state passed to every phase method.
type Context struct {
	// ConfigRepoPath is the absolute path to the cloned config repo on disk.
	// Module implementations read source files relative to this path.
	ConfigRepoPath string

	// Identity holds the user identity fields from sproot.yaml.
	Identity config.Identity

	// Log is the structured logger. Phases should use this for all output.
	Log *log.Logger

	// DryRun, when true, instructs phases to describe their actions without
	// executing any side effects.
	DryRun bool

	// Force, when true, means the runner will call Run regardless of what
	// ShouldRun returns.
	Force bool

	// OnlyFilter is non-empty when --only was passed. It equals the phase type
	// that was selected. The verify phase uses this to skip checks that are only
	// relevant to phases that were filtered out.
	OnlyFilter string
}
