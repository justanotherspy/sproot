package modules

import (
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestSpriteService_TypeAndName(t *testing.T) {
	p := &spriteServicePhase{cfg: &config.SpriteServiceConfig{
		Service: "dockerd",
		Cmd:     "/usr/bin/dockerd",
	}}
	if p.Type() != "sprite_service" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "sprite_service(dockerd)" {
		t.Errorf("Name: got %q", p.Name())
	}
}

func TestSpriteService_ShouldRunWhenSpriteEnvMissing(t *testing.T) {
	// sprite-env is not present in dev environments; ShouldRun returns true.
	p := &spriteServicePhase{cfg: &config.SpriteServiceConfig{
		Service: "dockerd",
		Cmd:     "/usr/bin/dockerd",
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when sprite-env not available")
	}
}
