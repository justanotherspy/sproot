package modules

import (
	"testing"
)

func TestGHToken_RunFailsWithoutToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	p := &ghTokenPhase{}
	if err := p.Run(testCtx(t)); err == nil {
		t.Error("expected error when GH_TOKEN not set")
	}
}

func TestGHToken_TypeAndName(t *testing.T) {
	p := &ghTokenPhase{}
	if p.Type() != "gh_token" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "gh_token" {
		t.Errorf("Name: got %q", p.Name())
	}
}

func TestGHToken_ShouldRunWhenGHNotAuth(t *testing.T) {
	// In dev environments gh is either absent or not logged in.
	// Either way, ShouldRun returns true.
	p := &ghTokenPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	// We can't control whether gh is authenticated in CI, so only
	// assert that if gh is not authenticated, ShouldRun returns true.
	if !checkCmd("gh", "auth", "status", "-h", "github.com") && !should {
		t.Error("expected ShouldRun=true when gh not authenticated")
	}
}

