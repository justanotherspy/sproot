// Package table renders box-drawing tables that can fill a target terminal
// width and apply ANSI styling. It is intentionally dependency-free so the
// single sproot binary stays lean.
package table

import (
	"io"
	"strings"
)

// Align controls horizontal text alignment within a column.
type Align int

const (
	// AlignLeft pads on the right.
	AlignLeft Align = iota
	// AlignRight pads on the left.
	AlignRight
	// AlignCenter pads on both sides.
	AlignCenter
)

// Style is an ANSI SGR sequence applied to a cell or border when Color is on.
type Style string

const (
	// StyleNone applies no styling.
	StyleNone Style = ""
	// StyleDim renders faint text (used for borders).
	StyleDim Style = "\x1b[2m"
	// StyleRed marks error states.
	StyleRed Style = "\x1b[31m"
	// StyleGreen marks healthy/running states.
	StyleGreen Style = "\x1b[32m"
	// StyleYellow marks transitional states.
	StyleYellow Style = "\x1b[33m"
	// StyleBoldCyan styles header text.
	StyleBoldCyan Style = "\x1b[1;36m"
)

const reset = "\x1b[0m"

// Cell is a single table cell: plain text plus an optional style. Width is
// always measured from Text, so styling never affects column sizing.
type Cell struct {
	Text  string
	Style Style
}

// Column describes a column header and the alignment used for its data cells.
type Column struct {
	Header string
	Align  Align
}

// Table renders Rows under Columns as a box-drawing grid.
type Table struct {
	Columns []Column
	Rows    [][]Cell
	// Flex is the index of the column that absorbs slack to reach Width, or a
	// negative value for no flexing.
	Flex int
	// Width is the desired total width including borders. Zero renders at the
	// natural content width.
	Width int
	// Color enables ANSI styling of headers, status cells, and borders.
	Color bool
}

const minFlexWidth = 8

// Render writes the table to w. Rows with fewer cells than columns are padded
// with empty cells; extra cells are ignored.
func (t *Table) Render(w io.Writer) error {
	n := len(t.Columns)
	if n == 0 {
		return nil
	}

	widths := t.naturalWidths()
	t.fit(widths)

	var b strings.Builder
	t.writeRule(&b, widths, "┌", "┬", "┐") // top
	t.writeHeader(&b, widths)
	t.writeRule(&b, widths, "├", "┼", "┤") // middle
	for _, row := range t.Rows {
		t.writeRow(&b, widths, row)
	}
	t.writeRule(&b, widths, "└", "┴", "┘") // bottom

	_, err := io.WriteString(w, b.String())
	return err
}

func (t *Table) naturalWidths() []int {
	widths := make([]int, len(t.Columns))
	for i, c := range t.Columns {
		widths[i] = runeLen(c.Header)
	}
	for _, row := range t.Rows {
		for i := range t.Columns {
			if i < len(row) {
				if l := runeLen(row[i].Text); l > widths[i] {
					widths[i] = l
				}
			}
		}
	}
	return widths
}

// fit grows or shrinks the flex column so the rendered width matches t.Width.
func (t *Table) fit(widths []int) {
	if t.Width <= 0 || t.Flex < 0 || t.Flex >= len(widths) {
		return
	}
	// Per column overhead: a leading "|" plus a space on each side of content.
	// The table also has one trailing "|". So total = sum(width+3) + 1.
	natural := 1
	for _, wd := range widths {
		natural += wd + 3
	}
	switch {
	case natural < t.Width:
		widths[t.Flex] += t.Width - natural
	case natural > t.Width:
		shrink := natural - t.Width
		if room := widths[t.Flex] - minFlexWidth; room > 0 {
			if shrink > room {
				shrink = room
			}
			widths[t.Flex] -= shrink
		}
	}
}

func (t *Table) writeRule(b *strings.Builder, widths []int, left, mid, right string) {
	var line strings.Builder
	line.WriteString(left)
	for i, wd := range widths {
		line.WriteString(strings.Repeat("─", wd+2))
		if i < len(widths)-1 {
			line.WriteString(mid)
		}
	}
	line.WriteString(right)
	b.WriteString(t.style(line.String(), StyleDim))
	b.WriteByte('\n')
}

func (t *Table) writeHeader(b *strings.Builder, widths []int) {
	cells := make([]Cell, len(t.Columns))
	aligns := make([]Align, len(t.Columns))
	for i, c := range t.Columns {
		cells[i] = Cell{Text: c.Header, Style: StyleBoldCyan}
		aligns[i] = AlignCenter
	}
	t.writeCells(b, widths, cells, aligns)
}

func (t *Table) writeRow(b *strings.Builder, widths []int, row []Cell) {
	cells := make([]Cell, len(t.Columns))
	aligns := make([]Align, len(t.Columns))
	for i := range t.Columns {
		aligns[i] = t.Columns[i].Align
		if i < len(row) {
			cells[i] = row[i]
		}
	}
	t.writeCells(b, widths, cells, aligns)
}

func (t *Table) writeCells(b *strings.Builder, widths []int, cells []Cell, aligns []Align) {
	bar := t.style("│", StyleDim)
	b.WriteString(bar)
	for i, wd := range widths {
		text := truncate(cells[i].Text, wd)
		padded := pad(text, wd, aligns[i])
		b.WriteByte(' ')
		b.WriteString(t.style(padded, cells[i].Style))
		b.WriteByte(' ')
		b.WriteString(bar)
	}
	b.WriteByte('\n')
}

func (t *Table) style(s string, st Style) string {
	if !t.Color || st == StyleNone {
		return s
	}
	return string(st) + s + reset
}

func pad(s string, width int, a Align) string {
	gap := width - runeLen(s)
	if gap <= 0 {
		return s
	}
	switch a {
	case AlignRight:
		return strings.Repeat(" ", gap) + s
	case AlignCenter:
		left := gap / 2
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", gap-left)
	default:
		return s + strings.Repeat(" ", gap)
	}
}

func truncate(s string, width int) string {
	if runeLen(s) <= width {
		return s
	}
	r := []rune(s)
	if width <= 3 {
		return string(r[:width])
	}
	return string(r[:width-3]) + "..."
}

func runeLen(s string) int { return len([]rune(s)) }
