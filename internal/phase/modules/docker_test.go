package modules

import (
	"os/exec"
	"testing"
)

func TestDocker_ShouldRunWhenDockerMissing(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		p := &dockerPhase{}
		should, err := p.ShouldRun(testCtx(t))
		if err != nil {
			t.Fatal(err)
		}
		if !should {
			t.Error("expected ShouldRun=true when docker not on PATH")
		}
		return
	}
	t.Skip("docker on PATH; skip missing-tool path")
}

func TestDocker_ShouldRunFalseWhenDockerPresent(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not on PATH")
	}
	p := &dockerPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when docker already installed")
	}
}

func TestDocker_TypeAndName(t *testing.T) {
	p := &dockerPhase{}
	if p.Type() != "docker" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "docker" {
		t.Errorf("Name: got %q", p.Name())
	}
}
