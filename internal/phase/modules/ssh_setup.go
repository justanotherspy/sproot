package modules

// ssh_setup configures the injected SSH key: sets permissions, adds github.com to
// known_hosts, derives the public key, and writes the allowed_signers file.
// The host CLI injects the private key before running setup; this module does not generate it.
//
//	- type: ssh_setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("ssh_setup", func(cfg config.PhaseConfig) (phase.Phase, error) {
		return &sshSetupPhase{}, nil
	})
}

type sshSetupPhase struct{}

func (p *sshSetupPhase) Type() string { return "ssh_setup" }
func (p *sshSetupPhase) Name() string { return "ssh_setup" }

func (p *sshSetupPhase) ShouldRun(_ *phase.Context) (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return true, nil
	}
	kh := filepath.Join(home, ".ssh", "known_hosts")
	return !checkCmd("ssh-keygen", "-F", "github.com", "-f", kh), nil
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
	if err := os.Chmod(key, 0o600); err != nil {
		return fmt.Errorf("ssh_setup: chmod key: %w", err)
	}
	ctx.Log.Infof("set permissions on %s", key)

	pub := key + ".pub"
	pubBytes, err := outputOf("ssh-keygen", "-y", "-f", key)
	if err != nil {
		return fmt.Errorf("ssh_setup: derive pubkey: %w", err)
	}
	if err := os.WriteFile(pub, []byte(pubBytes+"\n"), 0o644); err != nil {
		return fmt.Errorf("ssh_setup: write pubkey: %w", err)
	}
	ctx.Log.Infof("wrote %s", pub)

	keyscanOut, err := outputOf("ssh-keyscan", "-H", "github.com")
	if err != nil {
		return fmt.Errorf("ssh_setup: ssh-keyscan: %w", err)
	}
	knownHosts := filepath.Join(sshDir, "known_hosts")
	f, err := os.OpenFile(knownHosts, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("ssh_setup: open known_hosts: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := fmt.Fprintln(f, keyscanOut); err != nil {
		return fmt.Errorf("ssh_setup: write known_hosts: %w", err)
	}
	ctx.Log.Info("appended github.com to known_hosts")

	if err := p.writeAllowedSigners(sshDir, pubBytes, ctx.Identity.GitUserEmail); err != nil {
		return err
	}
	return nil
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
	kh := filepath.Join(home, ".ssh", "known_hosts")
	if !checkCmd("ssh-keygen", "-F", "github.com", "-f", kh) {
		return fmt.Errorf("ssh_setup: github.com not in known_hosts")
	}
	pub := filepath.Join(home, ".ssh", "id_ed25519.pub")
	if _, err := os.Stat(pub); err != nil {
		return fmt.Errorf("ssh_setup: pubkey missing at %s", pub)
	}
	return nil
}
