package phase

import (
	"fmt"
	"maps"
	"slices"

	"github.com/justanotherspy/sproot/internal/config"
)

// registry maps phase type strings to factory functions.
// Populated by module init() calls via Register.
var registry = map[string]FactoryFunc{}

// Register adds a factory for the given phase type. It panics if the type is
// already registered (programming error). Called from init() in each module file.
func Register(phaseType string, fn FactoryFunc) {
	if _, exists := registry[phaseType]; exists {
		panic("phase type already registered: " + phaseType)
	}
	registry[phaseType] = fn
}

// Build looks up the factory for cfg.Type and returns a constructed Phase.
// Returns an error if no factory is registered for that type.
func Build(cfg config.PhaseConfig) (Phase, error) {
	fn, ok := registry[cfg.Type]
	if !ok {
		return nil, fmt.Errorf("no factory registered for phase type %q", cfg.Type)
	}
	return fn(cfg)
}

// Registered returns the sorted set of registered type strings, for diagnostics.
func Registered() []string {
	return slices.Sorted(maps.Keys(registry))
}
