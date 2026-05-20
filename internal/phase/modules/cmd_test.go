package modules

import (
	"io"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

func testCtx(t *testing.T) *phase.Context {
	t.Helper()
	return &phase.Context{
		ConfigRepoPath: t.TempDir(),
		Log:            log.New(io.Discard),
		Identity: config.Identity{
			GitUserName:      "Test User",
			GitUserEmail:     "test@example.com",
			GitDefaultBranch: "main",
			GHUsername:       "testuser",
		},
	}
}

func TestCmd_AlwaysRunsWithNoCheck(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true"}}
	ctx := testCtx(t)

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when no check is set")
	}
}

func TestCmd_SkipsWhenCheckPasses(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true", Check: "true"}}
	ctx := testCtx(t)

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when check exits 0")
	}
}

func TestCmd_RunsWhenCheckFails(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true", Check: "false"}}
	ctx := testCtx(t)

	should, err := p.ShouldRun(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when check exits non-zero")
	}
}

func TestCmd_RunSucceeds(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true"}}
	if err := p.Run(testCtx(t)); err != nil {
		t.Fatalf("Run: %v", err)
	}
}

func TestCmd_RunFailure(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "false"}}
	if err := p.Run(testCtx(t)); err == nil {
		t.Fatal("expected error from failing command")
	}
}

func TestCmd_VerifyNoCheck(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true"}}
	if err := p.Verify(testCtx(t)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestCmd_VerifyCheckPasses(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true", Check: "true"}}
	if err := p.Verify(testCtx(t)); err != nil {
		t.Fatalf("Verify: %v", err)
	}
}

func TestCmd_VerifyCheckFails(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true", Check: "false"}}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Fatal("expected Verify error when check fails")
	}
}

func TestCmd_NamedPhase(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true", Name: "mything"}}
	if got := p.Name(); got != "cmd(mything)" {
		t.Errorf("Name() = %q, want %q", got, "cmd(mything)")
	}
}

func TestCmd_UnnamedPhase(t *testing.T) {
	p := &cmdPhase{cfg: &config.CmdConfig{Run: "true"}}
	if got := p.Name(); got != "cmd" {
		t.Errorf("Name() = %q, want %q", got, "cmd")
	}
}
