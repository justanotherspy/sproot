package table

import (
	"regexp"
	"strings"
	"testing"
)

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func strip(s string) string { return ansi.ReplaceAllString(s, "") }

func sample(width int, color bool) *Table {
	return &Table{
		Columns: []Column{
			{Header: "NAME", Align: AlignLeft},
			{Header: "STATUS", Align: AlignLeft},
			{Header: "URL", Align: AlignLeft},
		},
		Flex:  2,
		Width: width,
		Color: color,
		Rows: [][]Cell{
			{{Text: "claude"}, {Text: "running", Style: StyleGreen}, {Text: "claude.sprites.app"}},
		},
	}
}

func TestRender_Structure(t *testing.T) {
	var b strings.Builder
	if err := sample(0, false).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	// top border, header, separator, one data row, bottom border.
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d:\n%s", len(lines), b.String())
	}
	if !strings.HasPrefix(lines[0], "┌") || !strings.HasSuffix(lines[0], "┐") {
		t.Errorf("bad top border: %q", lines[0])
	}
	if !strings.HasPrefix(lines[4], "└") || !strings.HasSuffix(lines[4], "┘") {
		t.Errorf("bad bottom border: %q", lines[4])
	}
	if !strings.Contains(lines[1], "NAME") || !strings.Contains(lines[1], "URL") {
		t.Errorf("header missing columns: %q", lines[1])
	}
	if !strings.Contains(lines[3], "claude") || !strings.Contains(lines[3], "running") {
		t.Errorf("data row missing values: %q", lines[3])
	}
}

func TestRender_FillsWidth(t *testing.T) {
	const width = 80
	var b strings.Builder
	if err := sample(width, false).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	for _, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if got := len([]rune(line)); got != width {
			t.Errorf("line width = %d, want %d: %q", got, width, line)
		}
	}
}

func TestRender_ShrinksFlexColumnWithEllipsis(t *testing.T) {
	tbl := &Table{
		Columns: []Column{{Header: "URL", Align: AlignLeft}},
		Flex:    0,
		Width:   12, // forces the long URL to be truncated
		Rows:    [][]Cell{{{Text: "a-very-long-sprite-url.sprites.app"}}},
	}
	var b strings.Builder
	if err := tbl.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(b.String(), "...") {
		t.Errorf("expected truncation ellipsis, got:\n%s", b.String())
	}
	for _, line := range strings.Split(strings.TrimRight(b.String(), "\n"), "\n") {
		if got := len([]rune(line)); got != 12 {
			t.Errorf("line width = %d, want 12: %q", got, line)
		}
	}
}

func TestRender_ColorWrapsStyledCells(t *testing.T) {
	var b strings.Builder
	if err := sample(0, true).Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := b.String()
	if !strings.Contains(out, string(StyleGreen)) {
		t.Errorf("expected green status styling in output")
	}
	if !strings.Contains(out, string(StyleBoldCyan)) {
		t.Errorf("expected bold cyan header styling in output")
	}
	// Stripping ANSI should leave the same logical content as the no-color render.
	var plain strings.Builder
	_ = sample(0, false).Render(&plain)
	if strip(out) != plain.String() {
		t.Errorf("stripped colored output differs from plain output:\n%q\n%q", strip(out), plain.String())
	}
}

func TestRender_RightAlign(t *testing.T) {
	tbl := &Table{
		Columns: []Column{{Header: "WHEN", Align: AlignRight}},
		Flex:    -1,
		Rows:    [][]Cell{{{Text: "ago"}}},
	}
	var b strings.Builder
	_ = tbl.Render(&b)
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	// "WHEN" is the widest (4); "ago" (3) is right aligned -> one leading space.
	if !strings.Contains(lines[3], "│  ago │") {
		t.Errorf("expected right-aligned cell, got %q", lines[3])
	}
}
