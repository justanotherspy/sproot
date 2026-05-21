package sprite

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/justanotherspy/sproot/internal/phase"
)

// PrintStatus loads the state file at statePath and writes a summary table to w.
// If no records are present (file absent or empty) a short message is printed instead.
func PrintStatus(statePath string, w io.Writer) error {
	state, err := phase.LoadState(statePath)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	return RenderStatus(state, statePath, w)
}

// RenderStatus writes a summary table for state to w. statePath is used only
// in the "no records" message; pass an empty string if rendering from memory.
func RenderStatus(state *phase.State, statePath string, w io.Writer) error {
	if len(state.Phases) == 0 {
		label := statePath
		if label == "" {
			label = "state"
		}
		_, err := fmt.Fprintf(w, "no phase records found at %s\n", label)
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	_, err := fmt.Fprintln(tw, "TYPE\tNAME\tSTATUS\tRAN AT\tERROR")
	if err != nil {
		return err
	}
	for _, rec := range state.Phases {
		errCol := truncate(rec.Error, 60)
		if errCol == "" {
			errCol = truncate(rec.VerifyError, 60)
		}
		_, err = fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			rec.Type,
			rec.Name,
			phaseStatusLabel(rec),
			rec.LastRunAt.Format("2006-01-02 15:04:05"),
			errCol,
		)
		if err != nil {
			return err
		}
	}
	return tw.Flush()
}

func phaseStatusLabel(rec phase.PhaseRecord) string {
	switch {
	case rec.Skipped:
		return "skipped"
	case rec.Error != "":
		return "failed"
	case rec.VerifyError != "":
		return "verify-failed"
	default:
		return "done"
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
