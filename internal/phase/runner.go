package phase

import (
	"errors"
	"fmt"
	"time"

	"github.com/justanotherspy/sproot/internal/config"
)

// RunnerOptions configures a Runner.
type RunnerOptions struct {
	// Only, if non-empty, restricts execution to phases whose Type() matches.
	// Phases of other types are omitted entirely (no state record written).
	Only string

	// StatePath is the path to write state.json. Defaults to the XDG config
	// path (~/.config/sproot/state.json on Linux) when empty.
	StatePath string
}

// Runner orchestrates execution of a list of Phase values.
type Runner struct {
	phases []Phase
	opts   RunnerOptions
}

// NewRunner builds Phase values from cfgs using the registry. Returns an error
// if any type has no registered factory.
func NewRunner(cfgs []config.PhaseConfig, opts RunnerOptions) (*Runner, error) {
	phases := make([]Phase, 0, len(cfgs))
	for _, cfg := range cfgs {
		p, err := Build(cfg)
		if err != nil {
			return nil, fmt.Errorf("build phase %q: %w", cfg.Type, err)
		}
		phases = append(phases, p)
	}
	return &Runner{phases: phases, opts: opts}, nil
}

// NewRunnerFromPhases constructs a Runner with pre-built phases. Used in tests
// and by callers that build Phase values directly. Production code should prefer
// NewRunner which handles registry lookup.
func NewRunnerFromPhases(phases []Phase, opts RunnerOptions) *Runner {
	return &Runner{phases: phases, opts: opts}
}

// Run executes all phases in order, writes the state file, and returns a joined
// error covering every phase failure. It continues past individual failures so
// all phases are attempted.
func (r *Runner) Run(ctx *Context) error {
	statePath := r.opts.StatePath
	if statePath == "" {
		p, err := DefaultStatePath()
		if err != nil {
			ctx.Log.Warnf("could not determine state path: %v", err)
			statePath = "state.json"
		} else {
			statePath = p
		}
	}

	state := &State{
		SchemaVersion: StateVersion,
		UpdatedAt:     now(),
	}
	var errs []error
	matched := 0

	for _, p := range r.phases {
		if r.opts.Only != "" && p.Type() != r.opts.Only {
			continue
		}
		matched++

		rec := PhaseRecord{
			Type:      p.Type(),
			Name:      p.Name(),
			LastRunAt: now(),
		}

		ctx.Log.Debugf("[%s] checking idempotency", p.Name())
		shouldRun, checkErr := p.ShouldRun(ctx)
		if checkErr != nil {
			rec.Error = checkErr.Error()
			ctx.Log.Errorf("[%s] idempotency check failed: %v", p.Name(), checkErr)
			state.Phases = append(state.Phases, rec)
			errs = append(errs, fmt.Errorf("phase %s: %w", p.Name(), checkErr))
			continue
		}
		ctx.Log.Debugf("[%s] shouldRun=%v", p.Name(), shouldRun)

		if !shouldRun && !ctx.Force {
			rec.Skipped = true
			ctx.Log.Infof("[%s] already done, skipping", p.Name())
			state.Phases = append(state.Phases, rec)
			continue
		}

		if !shouldRun && ctx.Force {
			rec.Forced = true
			ctx.Log.Warnf("[%s] forcing run", p.Name())
		}

		if ctx.DryRun {
			ctx.Log.Infof("[%s] dry-run: would run", p.Name())
			state.Phases = append(state.Phases, rec)
			continue
		}

		ctx.Log.Debugf("[%s] running", p.Name())
		if runErr := p.Run(ctx); runErr != nil {
			rec.Error = runErr.Error()
			ctx.Log.Errorf("[%s] failed: %v", p.Name(), runErr)
			state.Phases = append(state.Phases, rec)
			errs = append(errs, fmt.Errorf("phase %s: %w", p.Name(), runErr))
			continue
		}

		rec.DidWork = true

		ctx.Log.Debugf("[%s] verifying", p.Name())
		if verifyErr := p.Verify(ctx); verifyErr != nil {
			rec.VerifyError = verifyErr.Error()
			ctx.Log.Errorf("[%s] verify failed: %v", p.Name(), verifyErr)
			errs = append(errs, fmt.Errorf("phase %s verify: %w", p.Name(), verifyErr))
		} else {
			ctx.Log.Successf("[%s] done", p.Name())
		}

		state.Phases = append(state.Phases, rec)
	}

	if r.opts.Only != "" && matched == 0 {
		ctx.Log.Warnf("no phases matched --only %q", r.opts.Only)
	}

	state.UpdatedAt = now()
	if saveErr := SaveState(statePath, state); saveErr != nil {
		ctx.Log.Warnf("could not write state file: %v", saveErr)
	}

	r.logSummary(ctx, state)
	return errors.Join(errs...)
}

func (r *Runner) logSummary(ctx *Context, state *State) {
	var succeeded, skipped, failed, didWork int
	for _, rec := range state.Phases {
		switch {
		case rec.Skipped:
			skipped++
		case rec.Error != "" || rec.VerifyError != "":
			failed++
		default:
			succeeded++
			if rec.DidWork {
				didWork++
			}
		}
	}
	if failed > 0 {
		ctx.Log.Errorf("%d phase(s) failed, %d succeeded (%d did work, %d skipped)",
			failed, succeeded, didWork, skipped)
	} else {
		ctx.Log.Successf("%d phase(s) succeeded (%d did work, %d skipped)",
			succeeded, didWork, skipped)
	}
}

// now returns the current UTC time truncated to seconds.
func now() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
