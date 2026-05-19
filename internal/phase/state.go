package phase

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StateVersion is the schema_version written to state.json.
const StateVersion = 1

// State is the top-level structure of the sproot state file.
type State struct {
	SchemaVersion int           `json:"schema_version"`
	UpdatedAt     time.Time     `json:"updated_at"`
	Phases        []PhaseRecord `json:"phases"`
}

// PhaseRecord records the outcome of one phase execution.
type PhaseRecord struct {
	Type        string    `json:"type"`
	Name        string    `json:"name"`
	LastRunAt   time.Time `json:"last_run_at"`
	DidWork     bool      `json:"did_work"`
	Skipped     bool      `json:"skipped"`
	Forced      bool      `json:"forced"`
	Error       string    `json:"error,omitempty"`
	VerifyError string    `json:"verify_error,omitempty"`
}

// DefaultStatePath returns the path to the sproot state file using the XDG
// config directory (typically ~/.config/sproot/state.json on Linux).
func DefaultStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config dir: %w", err)
	}
	return filepath.Join(dir, "sproot", "state.json"), nil
}

// LoadState reads and parses the state file at path. If the file does not
// exist it returns an empty State (not an error).
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &State{SchemaVersion: StateVersion}, nil
		}
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	return &s, nil
}

// SaveState writes s to path as indented JSON, creating parent directories as
// needed. The file is written with mode 0640.
func SaveState(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("create state dir: %w", err)
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}
