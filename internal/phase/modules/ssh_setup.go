package modules

// ssh_setup generates a fresh ed25519 keypair (if absent), adds github.com to
// known_hosts, writes the allowed_signers file, and registers the public key
// with GitHub as both an authentication key and a signing key using GH_TOKEN.
//
// Key IDs returned by GitHub are logged so they can be used by sproot destroy
// to remove the keys from the account (Phase 5 work).
//
//	- type: ssh_setup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

// GHKeyIDs holds the GitHub key IDs registered by ssh_setup, written to disk
// so that sproot destroy can remove them from the account on cleanup.
type GHKeyIDs struct {
	AuthKeyID    int64 `json:"auth_key_id"`
	SigningKeyID int64 `json:"signing_key_id"`
}

var ghAPIClient = &http.Client{Timeout: 30 * time.Second}

func init() {
	phase.Register("ssh_setup", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &sshSetupPhase{ghRegister: registerKeyWithGitHub}, nil
	})
}

type sshSetupPhase struct {
	// ghRegister posts pubKey to GitHub as both an auth key and a signing key.
	// Returns (authKeyID, signingKeyID). Swapped in tests to avoid network calls.
	ghRegister func(token, title, pubKey string) (int64, int64, error)
}

func (p *sshSetupPhase) Type() string { return "ssh_setup" }
func (p *sshSetupPhase) Name() string { return "ssh_setup" }

func (p *sshSetupPhase) ShouldRun(_ *phase.Context) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return true, nil
	}
	key := filepath.Join(home, ".ssh", "id_ed25519")
	if _, err := os.Stat(key); os.IsNotExist(err) {
		return true, nil
	}
	kh := filepath.Join(home, ".ssh", "known_hosts")
	if !checkCmd("ssh-keygen", "-F", "github.com", "-f", kh) {
		return true, nil
	}
	// Return true if the local pubkey is missing from allowed_signers.
	pubBytes, err := os.ReadFile(key + ".pub")
	if err != nil {
		return true, nil
	}
	pubKey := strings.TrimSpace(string(pubBytes))
	signersPath := filepath.Join(home, ".ssh", "allowed_signers")
	signersData, err := os.ReadFile(signersPath)
	if err != nil || !strings.Contains(string(signersData), pubKey) {
		return true, nil
	}
	// Return true if GH_TOKEN is set but GitHub keys were never registered.
	if os.Getenv("GH_TOKEN") != "" {
		keysPath, err := GHKeyIDsPath()
		if err != nil {
			return true, nil
		}
		if _, err := os.Stat(keysPath); os.IsNotExist(err) {
			return true, nil
		}
	}
	return false, nil
}

func (p *sshSetupPhase) Run(ctx *phase.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return fmt.Errorf("ssh_setup: mkdir .ssh: %w", err)
	}

	key := filepath.Join(sshDir, "id_ed25519")
	pub := key + ".pub"

	if _, err := os.Stat(key); os.IsNotExist(err) {
		if err := exec.Command("ssh-keygen", "-t", "ed25519", "-f", key, "-N", "").Run(); err != nil {
			return fmt.Errorf("ssh_setup: generate key: %w", err)
		}
		ctx.Log.Info("generated ed25519 keypair")
	}

	if err := os.Chmod(key, 0o600); err != nil {
		return fmt.Errorf("ssh_setup: chmod key: %w", err)
	}

	pubBytes, err := os.ReadFile(pub)
	if err != nil {
		return fmt.Errorf("ssh_setup: read pubkey: %w", err)
	}
	pubKey := strings.TrimSpace(string(pubBytes))

	token := os.Getenv("GH_TOKEN")
	if token == "" {
		ctx.Log.Warn("GH_TOKEN not set; skipping GitHub key registration")
	} else {
		title := "sproot-" + ctx.Identity.GHUsername
		authID, signingID, err := p.ghRegister(token, title, pubKey)
		if err != nil {
			return fmt.Errorf("ssh_setup: GitHub key registration: %w", err)
		}
		ctx.Log.Infof("registered GitHub auth key (id=%d)", authID)
		ctx.Log.Infof("registered GitHub signing key (id=%d)", signingID)
		if authID != 0 || signingID != 0 {
			if err := writeKeyIDs(GHKeyIDs{AuthKeyID: authID, SigningKeyID: signingID}); err != nil {
				ctx.Log.Warnf("ssh_setup: could not persist key IDs: %v", err)
			}
		}
	}

	knownHosts := filepath.Join(sshDir, "known_hosts")
	// Only scan and append when github.com is not already present, so re-runs
	// (e.g. when the key exists but was never registered) do not accumulate
	// duplicate hashed known_hosts entries.
	if checkCmd("ssh-keygen", "-F", "github.com", "-f", knownHosts) {
		ctx.Log.Info("github.com already in known_hosts")
	} else {
		keyscanOut, err := outputOf("ssh-keyscan", "-H", "github.com")
		if err != nil {
			return fmt.Errorf("ssh_setup: ssh-keyscan: %w", err)
		}
		f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			return fmt.Errorf("ssh_setup: open known_hosts: %w", err)
		}
		defer func() { _ = f.Close() }()
		if _, err := fmt.Fprintln(f, keyscanOut); err != nil {
			return fmt.Errorf("ssh_setup: write known_hosts: %w", err)
		}
		ctx.Log.Info("appended github.com to known_hosts")
	}

	return p.writeAllowedSigners(sshDir, pubKey, ctx.Identity.GitUserEmail)
}

func (p *sshSetupPhase) writeAllowedSigners(sshDir, pubKey, email string) error {
	signers := filepath.Join(sshDir, "allowed_signers")
	entry := fmt.Sprintf("%s namespaces=\"git\" %s\n", email, pubKey)

	existing, _ := os.ReadFile(signers)
	if strings.Contains(string(existing), pubKey) {
		return nil
	}
	f, err := os.OpenFile(signers, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("ssh_setup: open allowed_signers: %w", err)
	}
	defer func() { _ = f.Close() }()
	_, err = f.WriteString(entry)
	return err
}

func (p *sshSetupPhase) Verify(_ *phase.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	key := filepath.Join(home, ".ssh", "id_ed25519")
	if _, err := os.Stat(key); err != nil {
		return fmt.Errorf("ssh_setup: private key missing at %s", key)
	}
	kh := filepath.Join(home, ".ssh", "known_hosts")
	if !checkCmd("ssh-keygen", "-F", "github.com", "-f", kh) {
		return fmt.Errorf("ssh_setup: github.com not in known_hosts")
	}
	pub := key + ".pub"
	if _, err := os.Stat(pub); err != nil {
		return fmt.Errorf("ssh_setup: pubkey missing at %s", pub)
	}
	return nil
}

// ghKeyResponse is the subset of GitHub's SSH key API response we need.
type ghKeyResponse struct {
	ID int64 `json:"id"`
}

// registerKeyWithGitHub posts pubKey to /user/keys and /user/ssh_signing_keys.
// A 422 from either endpoint means the key is already registered; ID 0 is
// returned for that slot rather than an error.
func registerKeyWithGitHub(token, title, pubKey string) (int64, int64, error) {
	authID, err := postGHKey(token, "https://api.github.com/user/keys", title, pubKey)
	if err != nil {
		return 0, 0, fmt.Errorf("auth key: %w", err)
	}
	signingID, err := postGHKey(token, "https://api.github.com/user/ssh_signing_keys", title, pubKey)
	if err != nil {
		return 0, 0, fmt.Errorf("signing key: %w", err)
	}
	return authID, signingID, nil
}

// GHKeyIDsPath returns the path where ssh_setup writes the registered key IDs.
func GHKeyIDsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("could not determine config dir: %w", err)
	}
	return filepath.Join(dir, "sproot", "github_keys.json"), nil
}

func writeKeyIDs(ids GHKeyIDs) error {
	path, err := GHKeyIDsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	data, err := json.MarshalIndent(ids, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal key IDs: %w", err)
	}
	return os.WriteFile(path, data, 0o640)
}

func postGHKey(token, url, title, pubKey string) (int64, error) {
	body, _ := json.Marshal(map[string]string{"title": title, "key": pubKey})
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := ghAPIClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusUnprocessableEntity {
		return 0, nil
	}
	if resp.StatusCode != http.StatusCreated {
		return 0, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result ghKeyResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("decode response: %w", err)
	}
	return result.ID, nil
}
