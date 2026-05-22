package phase

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/pkg/log"
)

// dummyPhase is a configurable Phase implementation for testing.
type dummyPhase struct {
	typ       string
	name      string
	shouldRun bool
	shouldErr error
	runErr    error
	verifyErr error
	ranRun    bool
	ranVerify bool
}

func (d *dummyPhase) Type() string                        { return d.typ }
func (d *dummyPhase) Name() string                        { return d.name }
func (d *dummyPhase) ShouldRun(_ *Context) (bool, error) { return d.shouldRun, d.shouldErr }
func (d *dummyPhase) Run(_ *Context) error                { d.ranRun = true; return d.runErr }
func (d *dummyPhase) Verify(_ *Context) error             { d.ranVerify = true; return d.verifyErr }

func newTestCtx(t *testing.T) *Context {
	t.Helper()
	return &Context{
		ConfigRepoPath: t.TempDir(),
		Log:            log.New(io.Discard),
	}
}

func newTestRunner(phases []Phase, statePath string) *Runner {
	return NewRunnerFromPhases(phases, RunnerOptions{StatePath: statePath})
}

func statePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "state.json")
}

func loadTestState(t *testing.T, path string) *State {
	t.Helper()
	s, err := LoadState(path)
	if err != nil {
		t.Fatalf("LoadState: %v", err)
	}
	return s
}

func TestRunner_LastState(t *testing.T) {
	r := newTestRunner([]Phase{&dummyPhase{typ: "ok", name: "ok", shouldRun: true}}, statePath(t))
	if r.LastState() != nil {
		t.Fatal("LastState should be nil before Run")
	}
	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	st := r.LastState()
	if st == nil {
		t.Fatal("LastState should be non-nil after Run")
	}
	if len(st.Phases) != 1 || st.Phases[0].Name != "ok" {
		t.Errorf("unexpected LastState phases: %+v", st.Phases)
	}
}

func TestRunner_PassPhaseRunsAndVerifies(t *testing.T) {
	p := &dummyPhase{typ: "ok", name: "ok", shouldRun: true}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ranRun {
		t.Error("Run was not called")
	}
	if !p.ranVerify {
		t.Error("Verify was not called")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 1 {
		t.Fatalf("want 1 phase record, got %d", len(s.Phases))
	}
	rec := s.Phases[0]
	if !rec.DidWork {
		t.Error("want DidWork=true")
	}
	if rec.Skipped {
		t.Error("want Skipped=false")
	}
	if rec.Error != "" {
		t.Errorf("want no Error, got %q", rec.Error)
	}
}

func TestRunner_SkipPhaseWhenShouldRunFalse(t *testing.T) {
	p := &dummyPhase{typ: "skip", name: "skip", shouldRun: false}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ranRun {
		t.Error("Run should not have been called")
	}
	if p.ranVerify {
		t.Error("Verify should not have been called")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 1 {
		t.Fatalf("want 1 phase record, got %d", len(s.Phases))
	}
	if !s.Phases[0].Skipped {
		t.Error("want Skipped=true")
	}
}

func TestRunner_FailPhaseReturnsError(t *testing.T) {
	p := &dummyPhase{typ: "fail", name: "fail", shouldRun: true, runErr: errors.New("boom")}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	err := r.Run(newTestCtx(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, p.runErr) {
		t.Errorf("error chain does not contain runErr: %v", err)
	}
	if p.ranVerify {
		t.Error("Verify should not be called when Run fails")
	}
	s := loadTestState(t, sp)
	if s.Phases[0].Error == "" {
		t.Error("want Error set in state record")
	}
}

func TestRunner_VerifyFailureRecordedAsError(t *testing.T) {
	p := &dummyPhase{typ: "apt", name: "apt", shouldRun: true, verifyErr: errors.New("not found")}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	err := r.Run(newTestCtx(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	s := loadTestState(t, sp)
	rec := s.Phases[0]
	if rec.VerifyError == "" {
		t.Error("want VerifyError set in state record")
	}
	if !rec.DidWork {
		t.Error("want DidWork=true even when Verify fails")
	}
}

func TestRunner_ForceOverridesSkip(t *testing.T) {
	p := &dummyPhase{typ: "force", name: "force", shouldRun: false}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)
	ctx := newTestCtx(t)
	ctx.Force = true

	if err := r.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p.ranRun {
		t.Error("Run should have been called when Force=true")
	}
	if !p.ranVerify {
		t.Error("Verify should have been called")
	}
	s := loadTestState(t, sp)
	rec := s.Phases[0]
	if !rec.Forced {
		t.Error("want Forced=true in state record")
	}
	if rec.Skipped {
		t.Error("want Skipped=false when forced")
	}
}

func TestRunner_OnlyFiltersPhases(t *testing.T) {
	p1 := &dummyPhase{typ: "apt", name: "apt", shouldRun: true}
	p2 := &dummyPhase{typ: "cmd", name: "cmd", shouldRun: true}
	sp := statePath(t)
	r := NewRunnerFromPhases([]Phase{p1, p2}, RunnerOptions{Only: "apt", StatePath: sp})

	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !p1.ranRun {
		t.Error("apt should have run")
	}
	if p2.ranRun {
		t.Error("cmd should not have run")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 1 {
		t.Errorf("want 1 phase record, got %d", len(s.Phases))
	}
	if s.Phases[0].Type != "apt" {
		t.Errorf("want apt record, got %q", s.Phases[0].Type)
	}
}

func TestRunner_ContinuesPastFailure(t *testing.T) {
	p1 := &dummyPhase{typ: "fail", name: "fail", shouldRun: true, runErr: errors.New("boom")}
	p2 := &dummyPhase{typ: "ok", name: "ok", shouldRun: true}
	sp := statePath(t)
	r := newTestRunner([]Phase{p1, p2}, sp)

	err := r.Run(newTestCtx(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !p2.ranRun {
		t.Error("second phase should run even after first fails")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 2 {
		t.Errorf("want 2 phase records, got %d", len(s.Phases))
	}
}

func TestRunner_DryRunSkipsExecution(t *testing.T) {
	p := &dummyPhase{typ: "apt", name: "apt", shouldRun: true}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)
	ctx := newTestCtx(t)
	ctx.DryRun = true

	if err := r.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ranRun {
		t.Error("Run should not be called in dry-run mode")
	}
	if p.ranVerify {
		t.Error("Verify should not be called in dry-run mode")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 1 {
		t.Fatalf("want 1 phase record, got %d", len(s.Phases))
	}
	if s.Phases[0].DidWork {
		t.Error("want DidWork=false in dry-run")
	}
}

func TestRunner_StateFileWritten(t *testing.T) {
	p := &dummyPhase{typ: "apt", name: "apt", shouldRun: true}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(sp); err != nil {
		t.Fatalf("state file not written: %v", err)
	}
	s := loadTestState(t, sp)
	if s.SchemaVersion != StateVersion {
		t.Errorf("want schema_version %d, got %d", StateVersion, s.SchemaVersion)
	}
	if len(s.Phases) != 1 || s.Phases[0].Type != "apt" {
		t.Errorf("unexpected phases in state: %+v", s.Phases)
	}
}

func TestRunner_OnlyNoMatch(t *testing.T) {
	p := &dummyPhase{typ: "apt", name: "apt", shouldRun: true}
	sp := statePath(t)
	r := NewRunnerFromPhases([]Phase{p}, RunnerOptions{Only: "bogus", StatePath: sp})

	if err := r.Run(newTestCtx(t)); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.ranRun {
		t.Error("phase should not have run")
	}
	s := loadTestState(t, sp)
	if len(s.Phases) != 0 {
		t.Errorf("want 0 phase records, got %d", len(s.Phases))
	}
}

func TestRunner_ShouldRunErrorRecorded(t *testing.T) {
	p := &dummyPhase{typ: "apt", name: "apt", shouldErr: errors.New("check failed")}
	sp := statePath(t)
	r := newTestRunner([]Phase{p}, sp)

	err := r.Run(newTestCtx(t))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if p.ranRun {
		t.Error("Run should not be called when ShouldRun errors")
	}
	s := loadTestState(t, sp)
	if s.Phases[0].Error == "" {
		t.Error("want Error in state record when ShouldRun fails")
	}
}
