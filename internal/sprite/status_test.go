package sprite

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/justanotherspy/sproot/internal/phase"
)

func TestPrintStatus_NoRecords(t *testing.T) {
	var buf bytes.Buffer
	// Non-existent path: LoadState returns empty State, not an error.
	if err := PrintStatus("/tmp/sproot-test-nonexistent-state.json", &buf); err != nil {
		t.Fatalf("PrintStatus: %v", err)
	}
	if !strings.Contains(buf.String(), "no phase records found") {
		t.Errorf("expected 'no phase records found', got: %q", buf.String())
	}
}

func TestPrintStatus_Table(t *testing.T) {
	// Write a temp state file with a mix of outcomes.
	tmpDir := t.TempDir()
	statePath := tmpDir + "/state.json"

	state := &phase.State{
		SchemaVersion: phase.StateVersion,
		UpdatedAt:     time.Now(),
		Phases: []phase.PhaseRecord{
			{Type: "apt", Name: "apt packages", DidWork: true, LastRunAt: time.Now()},
			{Type: "go_install", Name: "go_install(golangci-lint)", Skipped: true, LastRunAt: time.Now()},
			{Type: "cmd", Name: "cmd", Error: "exit status 1", LastRunAt: time.Now()},
			{Type: "file_template", Name: "file_template(~/.claude/statusline.py)", VerifyError: "file missing", LastRunAt: time.Now()},
		},
	}
	if err := phase.SaveState(statePath, state); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintStatus(statePath, &buf); err != nil {
		t.Fatalf("PrintStatus: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"apt", "done",
		"go_install", "skipped",
		"cmd", "failed", "exit status 1",
		"file_template", "verify-failed", "file missing",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in output:\n%s", want, output)
		}
	}
}

func TestPrintStatus_ErrorTruncation(t *testing.T) {
	tmpDir := t.TempDir()
	statePath := tmpDir + "/state.json"

	longErr := strings.Repeat("x", 80)
	state := &phase.State{
		SchemaVersion: phase.StateVersion,
		UpdatedAt:     time.Now(),
		Phases: []phase.PhaseRecord{
			{Type: "cmd", Name: "cmd", Error: longErr, LastRunAt: time.Now()},
		},
	}
	if err := phase.SaveState(statePath, state); err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := PrintStatus(statePath, &buf); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if strings.Contains(output, longErr) {
		t.Error("expected long error to be truncated")
	}
	if !strings.Contains(output, "...") {
		t.Error("expected truncated error to end with '...'")
	}
}
