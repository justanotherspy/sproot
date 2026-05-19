// Package phase defines the Phase interface, runner, and shared execution context.
package phase

import "github.com/justanotherspy/sproot/internal/config"

// Phase is the interface every module must implement.
type Phase interface {
	// Type returns the module type string matching the sproot.yaml "type:" field.
	Type() string

	// Name returns a human-readable label used in log output. For simple modules
	// it may equal Type(). For parameterized modules it should include a
	// disambiguator (e.g. "binary_release(cosign)").
	Name() string

	// ShouldRun returns true if the phase has work to do. Returning false means
	// the environment is already in the target state and the phase can be skipped.
	// An error means the idempotency check itself failed.
	ShouldRun(ctx *Context) (bool, error)

	// Run executes the phase. Called when ShouldRun returns true, or when the
	// runner is operating with Force=true.
	Run(ctx *Context) error

	// Verify confirms the phase achieved its intended outcome. Called automatically
	// after every successful Run. An error here marks the phase as failed even
	// though Run succeeded.
	Verify(ctx *Context) error
}

// FactoryFunc constructs a Phase from a PhaseConfig.
type FactoryFunc func(cfg config.PhaseConfig) (Phase, error)
