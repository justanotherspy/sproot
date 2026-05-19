package phase

import (
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestBuild_UnknownType(t *testing.T) {
	_, err := Build(config.PhaseConfig{Type: "no-such-type-xyzzy"})
	if err == nil {
		t.Fatal("expected error for unknown type, got nil")
	}
	if !strings.Contains(err.Error(), "no-such-type-xyzzy") {
		t.Errorf("error should mention the type, got: %v", err)
	}
}

func TestRegister_DuplicatePanics(t *testing.T) {
	const typ = "test-dup-xyzzy"
	t.Cleanup(func() { delete(registry, typ) })

	Register(typ, func(_ config.PhaseConfig) (Phase, error) { return nil, nil })

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on duplicate registration, got none")
		}
	}()
	Register(typ, func(_ config.PhaseConfig) (Phase, error) { return nil, nil })
}

func TestRegistered_ReturnsSorted(t *testing.T) {
	const typ1 = "zzz-test-reg"
	const typ2 = "aaa-test-reg"
	t.Cleanup(func() {
		delete(registry, typ1)
		delete(registry, typ2)
	})

	Register(typ1, func(_ config.PhaseConfig) (Phase, error) { return nil, nil })
	Register(typ2, func(_ config.PhaseConfig) (Phase, error) { return nil, nil })

	names := Registered()
	// Find the two test entries and verify they are ordered.
	var idx1, idx2 int
	for i, n := range names {
		if n == typ1 {
			idx1 = i
		}
		if n == typ2 {
			idx2 = i
		}
	}
	// aaa... should come before zzz...
	if idx2 >= idx1 {
		t.Errorf("Registered() not sorted: %q at %d, %q at %d", typ2, idx2, typ1, idx1)
	}
}
