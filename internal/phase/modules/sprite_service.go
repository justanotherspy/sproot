package modules

// sprite_service registers a managed service with the sprite-env runtime via its
// internal Unix socket API. The service is started and kept alive by sprite-env.
//
//	- type: sprite_service
//	  service: dockerd
//	  cmd: /usr/bin/dockerd
//	  args: []   # optional

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("sprite_service", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.SpriteService == nil {
			return nil, fmt.Errorf("sprite_service: config is nil")
		}
		return &spriteServicePhase{cfg: cfg.SpriteService}, nil
	})
}

type spriteServicePhase struct {
	cfg *config.SpriteServiceConfig
}

func (p *spriteServicePhase) Type() string { return "sprite_service" }
func (p *spriteServicePhase) Name() string {
	return fmt.Sprintf("sprite_service(%s)", p.cfg.Service)
}

func (p *spriteServicePhase) ShouldRun(_ *phase.Context) (bool, error) {
	return !checkCmd("sprite-env", "curl", "/v1/services/"+p.cfg.Service), nil
}

func (p *spriteServicePhase) Run(ctx *phase.Context) error {
	body := map[string]any{"cmd": p.cfg.Cmd}
	if len(p.cfg.Args) > 0 {
		body["args"] = p.cfg.Args
	}
	bodyJSON, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("sprite_service: marshal body: %w", err)
	}
	return runCmd(ctx.Log,
		"sprite-env", "curl",
		"-X", "PUT",
		"/v1/services/"+p.cfg.Service,
		"-d", string(bodyJSON),
	)
}

func (p *spriteServicePhase) Verify(_ *phase.Context) error {
	out, err := outputOf("sprite-env", "curl", "/v1/services/"+p.cfg.Service)
	if err != nil {
		return fmt.Errorf("sprite_service: service %q not registered: %w", p.cfg.Service, err)
	}
	if !strings.Contains(out, p.cfg.Service) && !strings.Contains(out, p.cfg.Cmd) {
		return fmt.Errorf("sprite_service: unexpected response for %q: %s", p.cfg.Service, out)
	}
	return nil
}
