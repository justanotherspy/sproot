package modules

import (
	"os/exec"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestUVTool_ShouldRunWhenBinaryMissing(t *testing.T) {
	if _, err := exec.LookPath("uv"); err != nil {
		t.Skip("uv not on PATH")
	}
	p := &uvToolPhase{cfg: &config.UVToolConfig{
		Tools: []config.UVTool{{Name: "sproot-nonexistent-tool-xyzzy"}},
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when tool binary not on PATH")
	}
}

func TestUVTool_VerifyFailsWhenBinaryMissing(t *testing.T) {
	p := &uvToolPhase{cfg: &config.UVToolConfig{
		Tools: []config.UVTool{{Name: "sproot-nonexistent-tool-xyzzy"}},
	}}
	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when tool not on PATH")
	}
}

func TestUVTool_TypeAndName(t *testing.T) {
	p := &uvToolPhase{cfg: &config.UVToolConfig{}}
	if p.Type() != "uv_tool" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "uv_tool" {
		t.Errorf("Name: got %q", p.Name())
	}
}
