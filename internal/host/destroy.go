package host

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase/modules"
	"github.com/justanotherspy/sproot/pkg/log"
)

// ghKeyIDsSpritePath is the absolute path inside a sprite where ssh_setup writes key IDs.
// Sprites run as root, so the XDG config dir resolves to /root/.config.
const ghKeyIDsSpritePath = "/root/.config/sproot/github_keys.json"

// DestroyOptions controls the behavior of RunDestroy.
type DestroyOptions struct {
	Name       string
	client     SpritesClient // nil: constructed from token at runtime
	ghAPIBase  string        // nil: "https://api.github.com" (test injection point)
}

// RunDestroy reads GitHub key IDs from the sprite, removes them from GitHub,
// then destroys the sprite.
func RunDestroy(ctx context.Context, opts DestroyOptions) error {
	l := log.Stderr()

	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return err
	}

	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}
	ghToken := os.Getenv(cfg.GHTokenEnv)

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	handle := client.GetHandle(opts.Name)

	if ghToken != "" {
		data, err := handle.ReadFile(ghKeyIDsSpritePath)
		switch {
		case os.IsNotExist(err):
			l.Info("no SSH keys registered in this sprite, skipping key cleanup")
		case err != nil:
			l.Warnf("could not read GitHub key IDs from sprite: %v", err)
		default:
			var ids modules.GHKeyIDs
			if err := json.Unmarshal(data, &ids); err != nil {
				l.Warnf("could not parse GitHub key IDs: %v", err)
			} else {
				base := opts.ghAPIBase
				if base == "" {
					base = "https://api.github.com"
				}
				deleteGHKey(l, ghToken, base+"/user/keys", ids.AuthKeyID)
				deleteGHKey(l, ghToken, base+"/user/ssh_signing_keys", ids.SigningKeyID)
			}
		}
	}

	if err := client.DestroySprite(ctx, opts.Name); err != nil {
		return fmt.Errorf("destroy sprite: %w", err)
	}
	l.Infof("destroyed %s", opts.Name)
	return nil
}

func deleteGHKey(l *log.Logger, token, baseURL string, id int64) {
	if id == 0 {
		return
	}
	url := fmt.Sprintf("%s/%d", baseURL, id)
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		l.Warnf("build delete request for key %d: %v", id, err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		l.Warnf("delete GitHub key %d: %v", id, err)
		return
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		l.Warnf("delete GitHub key %d: unexpected status %d", id, resp.StatusCode)
	} else {
		l.Infof("deleted GitHub key %d", id)
	}
}
