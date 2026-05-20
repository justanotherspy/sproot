package modules

import (
	"testing"
)

func TestGHToken_TypeAndName(t *testing.T) {
	p := &ghTokenPhase{}
	if p.Type() != "gh_token" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "gh_token" {
		t.Errorf("Name: got %q", p.Name())
	}
}

func TestGHToken_RunWithoutTokenSkips(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	p := &ghTokenPhase{}
	if err := p.Run(testCtx(t)); err != nil {
		t.Errorf("expected nil when GH_TOKEN not set, got: %v", err)
	}
}

func TestGHToken_ShouldRunSkipsWithoutToken(t *testing.T) {
	t.Setenv("GH_TOKEN", "")
	p := &ghTokenPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when GH_TOKEN not set")
	}
}

func TestGHToken_ShouldRunWhenGHNotAuth(t *testing.T) {
	t.Setenv("GH_TOKEN", "some-token")
	p := &ghTokenPhase{}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	// Can't control whether gh is authenticated in this environment, so only
	// assert that if gh is not authenticated, ShouldRun returns true.
	if !checkCmd("gh", "auth", "status", "-h", "github.com") && !should {
		t.Error("expected ShouldRun=true when GH_TOKEN set but gh not authenticated")
	}
}

