package log_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/pkg/log"
)

func TestLogger_Prefixes(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(l *log.Logger)
		prefix string
	}{
		{"Info", func(l *log.Logger) { l.Info("hello") }, "- hello"},
		{"Success", func(l *log.Logger) { l.Success("done") }, "+ done"},
		{"Warn", func(l *log.Logger) { l.Warn("careful") }, "! careful"},
		{"Error", func(l *log.Logger) { l.Error("oops") }, "x oops"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := log.New(&buf)
			tt.fn(l)
			got := strings.TrimRight(buf.String(), "\n")
			if got != tt.prefix {
				t.Errorf("got %q, want %q", got, tt.prefix)
			}
		})
	}
}

func TestLogger_FormatVariants(t *testing.T) {
	var buf bytes.Buffer
	l := log.New(&buf)

	l.Infof("count %d", 3)
	if !strings.HasPrefix(buf.String(), "- count 3") {
		t.Errorf("Infof: got %q", buf.String())
	}
	buf.Reset()

	l.Successf("done %s", "ok")
	if !strings.HasPrefix(buf.String(), "+ done ok") {
		t.Errorf("Successf: got %q", buf.String())
	}
	buf.Reset()

	l.Warnf("retry %d", 1)
	if !strings.HasPrefix(buf.String(), "! retry 1") {
		t.Errorf("Warnf: got %q", buf.String())
	}
	buf.Reset()

	l.Errorf("failed: %s", "boom")
	if !strings.HasPrefix(buf.String(), "x failed: boom") {
		t.Errorf("Errorf: got %q", buf.String())
	}
}
