package host

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRenderList_PlainIsTabSeparated(t *testing.T) {
	created := time.Date(2026, 5, 1, 12, 0, 0, 0, time.UTC)
	var buf bytes.Buffer
	renderList(&buf, []SpriteListEntry{
		{Name: "claude", Status: "running", URL: "claude.sprites.app", CreatedAt: created},
	}, false, 0)

	out := buf.String()
	if strings.ContainsAny(out, "│┌┐└┘") {
		t.Errorf("plain output should not contain box characters: %q", out)
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("plain output should not contain ANSI codes: %q", out)
	}
	want := "claude\trunning\tclaude.sprites.app\t2026-05-01T12:00:00Z\t\n"
	if out != want {
		t.Errorf("plain output = %q, want %q", out, want)
	}
}

func TestRenderList_FancyDrawsBoxAndSummary(t *testing.T) {
	var buf bytes.Buffer
	renderList(&buf, []SpriteListEntry{
		{Name: "claude", Status: "running", URL: "claude.sprites.app", Organization: "acme", CreatedAt: time.Now().Add(-2 * time.Hour)},
	}, true, 80)

	out := buf.String()
	if !strings.Contains(out, "┌") || !strings.Contains(out, "┘") {
		t.Errorf("fancy output should draw a box: %q", out)
	}
	if !strings.Contains(out, "acme: 1 running / 1 total") {
		t.Errorf("fancy output missing summary: %q", out)
	}
	if !strings.Contains(out, string("\x1b[32m")) { // running -> green
		t.Errorf("fancy output should color the running status: %q", out)
	}
	if !strings.Contains(out, "NAME") || !strings.Contains(out, "LAST RUNNING") {
		t.Errorf("fancy output missing headers: %q", out)
	}
}

func TestListSummary_OrgPrefixOnlyWhenUniform(t *testing.T) {
	mixed := listSummary([]SpriteListEntry{
		{Status: "running", Organization: "acme"},
		{Status: "cold", Organization: "globex"},
	})
	if mixed != "1 running / 2 total" {
		t.Errorf("mixed orgs summary = %q", mixed)
	}
	uniform := listSummary([]SpriteListEntry{
		{Status: "running", Organization: "acme"},
		{Status: "running", Organization: "acme"},
	})
	if uniform != "acme: 2 running / 2 total" {
		t.Errorf("uniform org summary = %q", uniform)
	}
}

func TestHumanizeSince(t *testing.T) {
	now := time.Now()
	cases := []struct {
		in   *time.Time
		want string
	}{
		{nil, ""},
		{&time.Time{}, ""},
		{ptr(now.Add(-30 * time.Second)), "just now"},
		{ptr(now.Add(-1 * time.Minute)), "1 minute ago"},
		{ptr(now.Add(-5 * time.Minute)), "5 minutes ago"},
		{ptr(now.Add(-1 * time.Hour)), "1 hour ago"},
		{ptr(now.Add(-4 * 24 * time.Hour)), "4 days ago"},
		{ptr(now.Add(-40 * 24 * time.Hour)), "1 month ago"},
		{ptr(now.Add(-800 * 24 * time.Hour)), "2 years ago"},
	}
	for _, c := range cases {
		if got := humanizeSince(c.in); got != c.want {
			t.Errorf("humanizeSince = %q, want %q", got, c.want)
		}
	}
}

func TestStatusStyle(t *testing.T) {
	if statusStyle("running") == "" {
		t.Error("running should be styled")
	}
	if statusStyle("RUNNING") == "" {
		t.Error("status matching should be case-insensitive")
	}
	if statusStyle("cold") != "" {
		t.Error("unknown status should be unstyled")
	}
}

func ptr(t time.Time) *time.Time { return &t }
