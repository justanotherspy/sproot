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

	if !strings.Contains(out, "# sproot setup summary") {
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

func TestWriteLLMContext_WritesSummaryAndPointer(t *testing.T) {
	dir := t.TempDir()
	if err := writeLLMContext(log.Stderr(), sampleState(), dir); err != nil {
		t.Fatalf("writeLLMContext: %v", err)
	}

	summaryPath := filepath.Join(dir, sprootSummaryRel)
	data, err := os.ReadFile(summaryPath)
	if err != nil {
		t.Fatalf("read %s: %v", summaryPath, err)
	}
	if !strings.Contains(string(data), "# sproot setup summary") {
		t.Errorf("%s missing expected header", summaryPath)
	}

	llm, err := os.ReadFile(filepath.Join(dir, "llm.txt"))
	if err != nil {
		t.Fatalf("read llm.txt: %v", err)
	}
	if !strings.Contains(string(llm), pointerBegin) || !strings.Contains(string(llm), pointerEnd) {
		t.Errorf("llm.txt missing pointer block:\n%s", llm)
	}
	if !strings.Contains(string(llm), summaryPath) {
		t.Errorf("llm.txt pointer does not reference summary path:\n%s", llm)
	}
}

// TestWriteLLMContext_PreservesPlatformContent ensures sproot appends to an
// existing platform llm.txt rather than overwriting it, and is idempotent
// across re-runs (no duplicate blocks).
func TestWriteLLMContext_PreservesPlatformContent(t *testing.T) {
	dir := t.TempDir()
	platform := "# Sprite Environment\n\nplatform-provided guidance\n"
	if err := os.WriteFile(filepath.Join(dir, "llm.txt"), []byte(platform), 0o644); err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 2; i++ {
		if err := writeLLMContext(log.Stderr(), sampleState(), dir); err != nil {
			t.Fatalf("writeLLMContext run %d: %v", i, err)
		}
	}

	llm, err := os.ReadFile(filepath.Join(dir, "llm.txt"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(llm)
	if !strings.Contains(s, "platform-provided guidance") {
		t.Errorf("platform content was clobbered:\n%s", s)
	}
	if n := strings.Count(s, pointerBegin); n != 1 {
		t.Errorf("expected exactly one pointer block after re-run, got %d:\n%s", n, s)
	}
}
