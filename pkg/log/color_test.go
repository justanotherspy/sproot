package log

import (
	"bytes"
	"strings"
	"testing"
)

// White-box test: the public New() only enables color for terminals, so we set
// the field directly to exercise the colored write path.

func TestWrite_ColorsSymbolOnly(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, color: true}

	l.Success("done")
	got := buf.String()
	want := ansiGreen + "+" + ansiReset + " done\n"
	if got != want {
		t.Errorf("Success colored: got %q, want %q", got, want)
	}
	// The message text must not be wrapped in color codes.
	if strings.Contains(strings.TrimSuffix(got, "\n")[len(ansiGreen+"+"+ansiReset):], "\x1b[") {
		t.Errorf("message text should be uncolored: %q", got)
	}
}

func TestWrite_InfoHasNoColorEvenWhenEnabled(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, color: true}
	l.Info("hello")
	if got := buf.String(); got != "- hello\n" {
		t.Errorf("Info should stay neutral: got %q", got)
	}
}

func TestWrite_PlainWhenColorDisabled(t *testing.T) {
	var buf bytes.Buffer
	l := &Logger{w: &buf, color: false}
	l.Error("oops")
	if got := buf.String(); got != "x oops\n" {
		t.Errorf("color disabled: got %q, want %q", got, "x oops\n")
	}
}
