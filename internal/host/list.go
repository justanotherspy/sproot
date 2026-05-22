package host

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
	"github.com/justanotherspy/sproot/pkg/table"
	"github.com/justanotherspy/sproot/pkg/tty"
)

// ListOptions controls the behavior of RunList.
type ListOptions struct {
	All    bool          // show all sprites, not just sproot-managed ones
	Prefix string        // filter sprites by name prefix (empty means no filter)
	Watch  bool          // poll and refresh every 2 seconds
	client SpritesClient // nil: constructed from token at runtime
}

// RunList lists sprites. Without --all, only sprites tagged with the sproot
// label (created by sproot new) are shown. --prefix filters by name prefix.
// --watch polls and refreshes every 2 seconds until the context is cancelled.
func RunList(ctx context.Context, opts ListOptions) error {
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

	if opts.Watch {
		for {
			fmt.Print("\033[H\033[2J")
			if err := printList(ctx, client, opts); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(2 * time.Second):
			}
		}
	}
	return printList(ctx, client, opts)
}

func printList(ctx context.Context, client SpritesClient, opts ListOptions) error {
	l := log.Stderr()

	entries, err := client.ListSprites(ctx)
	if err != nil {
		return fmt.Errorf("list sprites: %w", err)
	}

	var shown []SpriteListEntry
	for _, e := range entries {
		if !opts.All && !hasLabel(e.Labels, labelBase) {
			continue
		}
		if opts.Prefix != "" && !strings.HasPrefix(e.Name, opts.Prefix) {
			continue
		}
		shown = append(shown, e)
		l.Debugf("labels: %v", e.Labels)
	}

	if len(shown) == 0 {
		if opts.All {
			l.Info("no sprites found")
		} else {
			l.Info("no sproot-managed sprites found (use --all to see all sprites)")
		}
		return nil
	}

	renderList(os.Stdout, shown, tty.IsColor(os.Stdout), tty.Width(os.Stdout))
	return nil
}

// renderList writes the sprite list to w. When fancy is true it draws a colored
// box-drawing table sized to width; otherwise it writes tab-separated rows that
// stay friendly to pipes and scripts.
func renderList(w io.Writer, entries []SpriteListEntry, fancy bool, width int) {
	if !fancy {
		for _, e := range entries {
			_, _ = fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				e.Name, e.Status, e.URL,
				formatStamp(&e.CreatedAt), formatStamp(e.LastRunningAt))
		}
		return
	}

	_, _ = fmt.Fprintln(w, listSummary(entries))

	t := table.Table{
		Columns: []table.Column{
			{Header: "NAME", Align: table.AlignLeft},
			{Header: "STATUS", Align: table.AlignLeft},
			{Header: "URL", Align: table.AlignLeft},
			{Header: "CREATED", Align: table.AlignRight},
			{Header: "LAST RUNNING", Align: table.AlignRight},
		},
		Flex:  2, // URL absorbs slack to fill the terminal width
		Width: width,
		Color: true,
	}
	for _, e := range entries {
		t.Rows = append(t.Rows, []table.Cell{
			{Text: e.Name},
			{Text: e.Status, Style: statusStyle(e.Status)},
			{Text: e.URL},
			{Text: humanizeSince(&e.CreatedAt)},
			{Text: humanizeSince(e.LastRunningAt)},
		})
	}
	_ = t.Render(w)
}

// listSummary mirrors the sprite CLI's header line: "<org>: <running> running /
// <total> total". The org prefix is shown only when every sprite shares one.
func listSummary(entries []SpriteListEntry) string {
	var running int
	org := ""
	uniform := true
	for i, e := range entries {
		if e.Status == "running" {
			running++
		}
		switch {
		case i == 0:
			org = e.Organization
		case e.Organization != org:
			uniform = false
		}
	}
	summary := fmt.Sprintf("%d running / %d total", running, len(entries))
	if uniform && org != "" {
		summary = org + ": " + summary
	}
	return summary
}

// statusStyle maps a sprite status to a table cell color.
func statusStyle(status string) table.Style {
	switch strings.ToLower(status) {
	case "running":
		return table.StyleGreen
	case "starting", "warming", "warm", "stopping", "suspending":
		return table.StyleYellow
	case "failed", "error", "destroyed":
		return table.StyleRed
	default:
		return table.StyleNone
	}
}

// humanizeSince renders t as a coarse "N units ago" string. A nil or zero time
// yields an empty string.
func humanizeSince(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	d := time.Since(*t)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return plural(int(d.Minutes()), "minute")
	case d < 24*time.Hour:
		return plural(int(d.Hours()), "hour")
	case d < 30*24*time.Hour:
		return plural(int(d.Hours()/24), "day")
	case d < 365*24*time.Hour:
		return plural(int(d.Hours()/(24*30)), "month")
	default:
		return plural(int(d.Hours()/(24*365)), "year")
	}
}

func plural(n int, unit string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s ago", unit)
	}
	return fmt.Sprintf("%d %ss ago", n, unit)
}

// formatStamp renders t as RFC3339 for machine-readable plain output, or empty
// for a nil/zero time.
func formatStamp(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

// hasLabel reports whether labels contains the target string.
func hasLabel(labels []string, target string) bool {
	for _, l := range labels {
		if l == target {
			return true
		}
	}
	return false
}
