// Package log provides structured logging using the +/-/!/x visual conventions.
// Logger is not safe for concurrent use.
package log

import (
	"fmt"
	"io"
	"os"
)

var debugEnabled bool

// SetDebug enables or disables debug-level output for all loggers.
func SetDebug(v bool) { debugEnabled = v }

// Logger writes structured log lines to an io.Writer.
type Logger struct {
	w io.Writer
}

// New returns a Logger that writes to w.
func New(w io.Writer) *Logger {
	return &Logger{w: w}
}

// Stderr returns a Logger that writes to os.Stderr.
func Stderr() *Logger {
	return New(os.Stderr)
}

// Info prints an informational line: "- <msg>"
func (l *Logger) Info(msg string) { l.write("- " + msg) }

// Infof prints a formatted informational line: "- <msg>"
func (l *Logger) Infof(format string, args ...any) { l.Info(fmt.Sprintf(format, args...)) }

// Success prints a success line: "+ <msg>"
func (l *Logger) Success(msg string) { l.write("+ " + msg) }

// Successf prints a formatted success line: "+ <msg>"
func (l *Logger) Successf(format string, args ...any) { l.Success(fmt.Sprintf(format, args...)) }

// Warn prints a warning line: "! <msg>"
func (l *Logger) Warn(msg string) { l.write("! " + msg) }

// Warnf prints a formatted warning line: "! <msg>"
func (l *Logger) Warnf(format string, args ...any) { l.Warn(fmt.Sprintf(format, args...)) }

// Error prints an error line: "x <msg>"
func (l *Logger) Error(msg string) { l.write("x " + msg) }

// Errorf prints a formatted error line: "x <msg>"
func (l *Logger) Errorf(format string, args ...any) { l.Error(fmt.Sprintf(format, args...)) }

// Debug prints a debug line: ". <msg>" when debug mode is enabled via SetDebug.
func (l *Logger) Debug(msg string) {
	if debugEnabled {
		l.write(". " + msg)
	}
}

// Debugf prints a formatted debug line when debug mode is enabled via SetDebug.
func (l *Logger) Debugf(format string, args ...any) { l.Debug(fmt.Sprintf(format, args...)) }

// write emits a single log line. Write errors are intentionally discarded;
// a logger that fails silently is preferable to one that panics mid-run.
func (l *Logger) write(line string) {
	_, _ = fmt.Fprintln(l.w, line)
}
