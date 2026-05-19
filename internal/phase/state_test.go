package phase

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadState_FileNotExist(t *testing.T) {
	s, err := LoadState(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.SchemaVersion != StateVersion {
		t.Errorf("want schema_version %d, got %d", StateVersion, s.SchemaVersion)
	}
	if len(s.Phases) != 0 {
		t.Errorf("want empty phases, got %d", len(s.Phases))
	}
}

func TestSaveAndLoadState_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "state.json")
	ts := time.Now().UTC().Truncate(time.Second)
	original := &State{
		SchemaVersion: StateVersion,
		UpdatedAt:     ts,
		Phases: []PhaseRecord{
			{Type: "apt", Name: "apt", LastRunAt: ts, DidWork: true},
			{Type: "cmd", Name: "cmd", LastRunAt: ts, Skipped: true},
			{Type: "fail", Name: "fail", LastRunAt: ts, Error: "boom"},
		},
	}

	if err := SaveState(path, original); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	loaded, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}

	if loaded.SchemaVersion != original.SchemaVersion {
		t.Errorf("schema_version: got %d, want %d", loaded.SchemaVersion, original.SchemaVersion)
	}
	if !loaded.UpdatedAt.Equal(original.UpdatedAt) {
		t.Errorf("updated_at: got %v, want %v", loaded.UpdatedAt, original.UpdatedAt)
	}
	if len(loaded.Phases) != len(original.Phases) {
		t.Fatalf("phases count: got %d, want %d", len(loaded.Phases), len(original.Phases))
	}
	for i, want := range original.Phases {
		got := loaded.Phases[i]
		if got.Type != want.Type {
			t.Errorf("phases[%d].Type: got %q, want %q", i, got.Type, want.Type)
		}
		if got.DidWork != want.DidWork {
			t.Errorf("phases[%d].DidWork: got %v, want %v", i, got.DidWork, want.DidWork)
		}
		if got.Skipped != want.Skipped {
			t.Errorf("phases[%d].Skipped: got %v, want %v", i, got.Skipped, want.Skipped)
		}
		if got.Error != want.Error {
			t.Errorf("phases[%d].Error: got %q, want %q", i, got.Error, want.Error)
		}
	}
}

func TestSaveState_CreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "dirs", "state.json")
	s := &State{SchemaVersion: StateVersion}
	if err := SaveState(path, s); err != nil {
		t.Fatalf("SaveState: %v", err)
	}
	if _, err := LoadState(path); err != nil {
		t.Fatalf("LoadState after SaveState: %v", err)
	}
}
