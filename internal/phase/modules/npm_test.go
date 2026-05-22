package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestNpm_ShouldRunWhenNodeModulesMissing(t *testing.T) {
	dir := t.TempDir()
	p := &npmPhase{cfg: &config.NpmConfig{Dir: dir}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when node_modules missing")
	}
}

func TestNpm_ShouldRunFalseWhenNodeModulesPresent(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)
	p := &npmPhase{cfg: &config.NpmConfig{Dir: dir}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when node_modules present")
	}
}

func TestNpm_VerifyFailsWhenNodeModulesMissing(t *testing.T) {
	dir := t.TempDir()
	p := &npmPhase{cfg: &config.NpmConfig{Dir: dir}}

	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when node_modules missing")
	}
}

func TestNpm_VerifyPassesWhenNodeModulesPresent(t *testing.T) {
	dir := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dir, "node_modules"), 0o755)
	p := &npmPhase{cfg: &config.NpmConfig{Dir: dir}}

	if err := p.Verify(testCtx(t)); err != nil {
		t.Errorf("expected Verify to pass, got: %v", err)
	}
}
