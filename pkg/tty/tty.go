// Package tty centralizes terminal detection so commands agree on when to emit
// colored, box-drawing output versus plain, script-friendly text.
package tty

import (
	"io"
	"os"

	"golang.org/x/term"
)

// IsColor reports whether colored output is appropriate for w. It must be a
// terminal-backed *os.File, NO_COLOR must be unset, and TERM must not be
// "dumb". Any non-file writer (buffer, pipe) yields false.
func IsColor(w io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return term.IsTerminal(int(f.Fd()))
}

// Width returns the column count of w when it is a terminal, or 0 otherwise.
func Width(w io.Writer) int {
	f, ok := w.(*os.File)
	if !ok {
		return 0
	}
	cols, _, err := term.GetSize(int(f.Fd()))
	if err != nil {
		return 0
	}
	return cols
}
