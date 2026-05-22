package host

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderOutdated_Plain(t *testing.T) {
	var buf bytes.Buffer
	renderOutdated(&buf, []outdatedRow{
		{Name: "api", Target: "(default)", SHA: "abc123", Status: "current"},
		{Name: "web", Target: "web", SHA: "def456", Status: "stale"},
	}, false, 0)

	out := buf.String()
	if strings.ContainsAny(out, "│┌┘") || strings.Contains(out, "\x1b[") {
		t.Errorf("plain output should be unstyled: %q", out)
	}
	if out != "api\t(default)\tabc123\tcurrent\nweb\tweb\tdef456\tstale\n" {
		t.Errorf("unexpected plain output: %q", out)
	}
}

func TestRenderOutdated_FancyColorsStatus(t *testing.T) {
	var buf bytes.Buffer
	renderOutdated(&buf, []outdatedRow{
		{Name: "api", Target: "(default)", SHA: "abc123", Status: "current"},
		{Name: "web", Target: "web", SHA: "def456", Status: "stale"},
	}, true, 80)

	out := buf.String()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "NAME") {
		t.Errorf("fancy output should draw a header box: %q", out)
	}
	if !strings.Contains(out, string("\x1b[32m")) { // current -> green
		t.Errorf("current status should be green: %q", out)
	}
	if !strings.Contains(out, string("\x1b[33m")) { // stale -> yellow
		t.Errorf("stale status should be yellow: %q", out)
	}
}

func TestRenderCheckpoints_Plain(t *testing.T) {
	ts, _ := time.Parse(time.RFC3339, "2026-05-21T10:00:00Z")
	var buf bytes.Buffer
	renderCheckpoints(&buf, []CheckpointEntry{
		{ID: "v1", CreateTime: ts, Comment: "first"},
		{ID: "v2", CreateTime: ts, Comment: "second", IsAuto: true},
	}, false, 0)

	out := buf.String()
	if strings.ContainsAny(out, "│┌┘") || strings.Contains(out, "\x1b[") {
		t.Errorf("plain output should be unstyled: %q", out)
	}
	if !strings.Contains(out, "v2\t2026-05-21 10:00:00\tsecond (auto)") {
		t.Errorf("auto suffix missing in plain output: %q", out)
	}
}

func TestRenderCheckpoints_FancyDrawsBox(t *testing.T) {
	ts, _ := time.Parse(time.RFC3339, "2026-05-21T10:00:00Z")
	var buf bytes.Buffer
	renderCheckpoints(&buf, []CheckpointEntry{
		{ID: "v1", CreateTime: ts, Comment: "first"},
	}, true, 80)

	out := buf.String()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "COMMENT") {
		t.Errorf("fancy output should draw a header box: %q", out)
	}
}
