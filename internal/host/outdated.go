package host

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
	"github.com/justanotherspy/sproot/pkg/table"
	"github.com/justanotherspy/sproot/pkg/tty"
)

// outdatedRow is one sprite's freshness compared to the current config SHA.
type outdatedRow struct {
	Name   string
	Target string
	SHA    string
	Status string
}

// OutdatedOptions controls the behavior of RunOutdated.
type OutdatedOptions struct {
	client SpritesClient
	shaFn  func() (string, ConfigMeta, error) // nil: compute from host config
}

// RunOutdated computes the current config SHA and compares it against every
// sproot-managed sprite's stored sproot-sha label. It prints a table showing
// whether each sprite is current or stale.
func RunOutdated(ctx context.Context, opts OutdatedOptions) error {
	l := log.Stderr()

	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return err
	}

	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	shaFn := opts.shaFn
	if shaFn == nil {
		shaFn = func() (string, ConfigMeta, error) {
			return currentConfigSHA(cfg, l)
		}
	}
	currentSHA, _, err := shaFn()
	if err != nil {
		return fmt.Errorf("compute current config SHA: %w", err)
	}

	all, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var rows []outdatedRow
	for _, e := range all {
		if !hasLabel(e.Labels, labelBase) {
			continue
		}
		meta := ParseConfigMeta(e.Labels)
		status := "current"
		if meta.SHA == "" {
			status = "unknown (no SHA label)"
		} else if meta.SHA != currentSHA {
			status = "stale"
		}
		target := meta.Target
		if target == "" {
			target = "(default)"
		}
		rows = append(rows, outdatedRow{Name: e.Name, Target: target, SHA: meta.SHA, Status: status})
	}

	if len(rows) == 0 {
		l.Info("no sproot-managed sprites found")
		return nil
	}

	renderOutdated(os.Stdout, rows, tty.IsColor(os.Stdout), tty.Width(os.Stdout))
	return nil
}

// renderOutdated writes the freshness table to w, drawing a colored box on a
// terminal and falling back to tab-separated rows when piped.
func renderOutdated(w io.Writer, rows []outdatedRow, color bool, width int) {
	if !color {
		for _, r := range rows {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", r.Name, r.Target, r.SHA, r.Status)
		}
		return
	}

	t := table.Table{
		Columns: []table.Column{
			{Header: "NAME", Align: table.AlignLeft},
			{Header: "TARGET", Align: table.AlignLeft},
			{Header: "SHA", Align: table.AlignLeft},
			{Header: "STATUS", Align: table.AlignLeft},
		},
		Flex:  0, // NAME absorbs slack to fill the terminal width
		Width: width,
		Color: true,
	}
	for _, r := range rows {
		t.Rows = append(t.Rows, []table.Cell{
			{Text: r.Name},
			{Text: r.Target},
			{Text: r.SHA},
			{Text: r.Status, Style: outdatedStyle(r.Status)},
		})
	}
	_ = t.Render(w)
}

func outdatedStyle(status string) table.Style {
	switch status {
	case "current":
		return table.StyleGreen
	case "stale":
		return table.StyleYellow
	default:
		return table.StyleRed
	}
}
