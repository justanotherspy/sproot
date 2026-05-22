package sprite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

func sampleState() *phase.State {
	return &phase.State{
		SchemaVersion: phase.StateVersion,
		UpdatedAt:     time.Date(2026, 5, 22, 12, 0, 0, 0, time.UTC),
		Phases: []phase.PhaseRecord{
			{Type: "apt", Name: "apt", DidWork: true},
			{Type: "npm", Name: "npm", DidWork: true},
			{Type: "git_identity", Name: "git_identity", Skipped: true},
			{Type: "ssh_setup", Name: "ssh_setup", Error: "boom"},
		},
	}
}

func TestRenderLLMContext_ListsWorkedModulesAndTally(t *testing.T) {
	out := renderLLMContext(sampleState())

	if !strings.Contains(out, "# Sprite environment") {
		t.Errorf("missing header in output:\n%s", out)
	}
	// Modules that did work should be described.
	if !strings.Contains(out, moduleDescriptions["apt"]) {
		t.Errorf("expected apt description in output:\n%s", out)
	}
	if !strings.Contains(out, moduleDescriptions["npm"]) {
		t.Errorf("expected npm description in output:\n%s", out)
	}
	// Skipped and failed modules should not appear in the provisioned list.
	if strings.Contains(out, moduleDescriptions["git_identity"]) {
		t.Errorf("did not expect skipped git_identity description in output:\n%s", out)
	}
	// Tally: 2 did work, 1 skipped, 1 failed.
	if !strings.Contains(out, "2 phase(s) did work, 1 already satisfied, 1 failed") {
		t.Errorf("unexpected tally in output:\n%s", out)
	}
}

func TestRenderLLMContext_NoWork(t *testing.T) {
	state := &phase.State{
		UpdatedAt: time.Now().UTC(),
		Phases: []phase.PhaseRecord{
			{Type: "apt", Name: "apt", Skipped: true},
		},
	}
	out := renderLLMContext(state)
	if !strings.Contains(out, "Nothing changed on this run") {
		t.Errorf("expected no-work message in output:\n%s", out)
	}
}

func TestWriteLLMContext_WritesBothFiles(t *testing.T) {
	dir := t.TempDir()
	if err := writeLLMContext(log.Stderr(), sampleState(), dir); err != nil {
		t.Fatalf("writeLLMContext: %v", err)
	}

	for _, rel := range []string{"llm.txt", filepath.Join("docs", "agent-context.md")} {
		path := filepath.Join(dir, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if len(data) == 0 {
			t.Errorf("%s is empty", path)
		}
		if !strings.Contains(string(data), "# Sprite environment") {
			t.Errorf("%s missing expected header", path)
		}
	}
}
