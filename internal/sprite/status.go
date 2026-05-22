package sprite

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/table"
	"github.com/justanotherspy/sproot/pkg/tty"
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

	if tty.IsColor(w) {
		return renderStatusTable(state, w, tty.Width(w))
	}
	return renderStatusPlain(state, w)
}

// renderStatusPlain writes the phase summary as a tabwriter-aligned table,
// the script-friendly fallback used when w is not a terminal.
func renderStatusPlain(state *phase.State, w io.Writer) error {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(tw, "TYPE\tNAME\tSTATUS\tRAN AT\tERROR"); err != nil {
		return err
	}
	for _, rec := range state.Phases {
		if _, err := fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n",
			rec.Type,
			rec.Name,
			phaseStatusLabel(rec),
			rec.LastRunAt.Format("2006-01-02 15:04:05"),
			statusErr(rec),
		); err != nil {
			return err
		}
	}
	return tw.Flush()
}

// renderStatusTable writes the phase summary as a colored box-drawing table.
func renderStatusTable(state *phase.State, w io.Writer, width int) error {
	t := table.Table{
		Columns: []table.Column{
			{Header: "TYPE", Align: table.AlignLeft},
			{Header: "NAME", Align: table.AlignLeft},
			{Header: "STATUS", Align: table.AlignLeft},
			{Header: "RAN AT", Align: table.AlignLeft},
			{Header: "ERROR", Align: table.AlignLeft},
		},
		Flex:  1, // NAME absorbs slack to fill the terminal width
		Width: width,
		Color: true,
	}
	for _, rec := range state.Phases {
		label := phaseStatusLabel(rec)
		t.Rows = append(t.Rows, []table.Cell{
			{Text: rec.Type},
			{Text: rec.Name},
			{Text: label, Style: phaseStatusStyle(label)},
			{Text: rec.LastRunAt.Format("2006-01-02 15:04:05")},
			{Text: statusErr(rec)},
		})
	}
	return t.Render(w)
}

// statusErr returns the phase error (or verify error) truncated for display.
func statusErr(rec phase.PhaseRecord) string {
	if e := truncate(rec.Error, 60); e != "" {
		return e
	}
	return truncate(rec.VerifyError, 60)
}

func phaseStatusStyle(label string) table.Style {
	switch label {
	case "done":
		return table.StyleGreen
	case "skipped":
		return table.StyleYellow
	case "failed", "verify-failed":
		return table.StyleRed
	default:
		return table.StyleNone
	}
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
