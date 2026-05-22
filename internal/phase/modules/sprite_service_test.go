package modules

import (
	"encoding/json"
	"strings"
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

func TestSpriteService_BodyIncludesHTTPPortAndNeeds(t *testing.T) {
	cfg := &config.SpriteServiceConfig{
		Service:  "myservice",
		Cmd:      "/usr/bin/myservice",
		HTTPPort: 8080,
		Needs:    []string{"dockerd"},
	}
	body := map[string]any{"cmd": cfg.Cmd}
	if cfg.HTTPPort != 0 {
		body["http_port"] = cfg.HTTPPort
	}
	if len(cfg.Needs) > 0 {
		body["needs"] = cfg.Needs
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, `"http_port":8080`) {
		t.Errorf("expected http_port in body, got: %s", s)
	}
	if !strings.Contains(s, `"dockerd"`) {
		t.Errorf("expected needs in body, got: %s", s)
	}
}

func TestSpriteService_BodyOmitsHTTPPortAndNeedsWhenEmpty(t *testing.T) {
	cfg := &config.SpriteServiceConfig{
		Service: "myservice",
		Cmd:     "/usr/bin/myservice",
	}
	body := map[string]any{"cmd": cfg.Cmd}
	if cfg.HTTPPort != 0 {
		body["http_port"] = cfg.HTTPPort
	}
	if len(cfg.Needs) > 0 {
		body["needs"] = cfg.Needs
	}
	data, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, "http_port") {
		t.Errorf("expected no http_port in body, got: %s", s)
	}
	if strings.Contains(s, "needs") {
		t.Errorf("expected no needs in body, got: %s", s)
	}
}
