package modules

// nix installs the Determinate Nix distribution, runs nix-daemon as a sprite
// service, declaratively installs packages into the user profile, wires the nix
// profile into the login shells, and optionally runs a setup script.
//
//	- type: nix
//	  packages:
//	    - hello                 # short form: nixpkgs#hello, symlinked as "hello"
//	    - name: ripgrep         # long form
//	      bin: rg               # binary differs from package name
//	    - name: nixfmt
//	      flake: nixpkgs#nixfmt-rfc-style  # explicit flake ref
//	      bin: nixfmt
//	  setup_script: files/nix-setup.sh     # optional; runs with the nix profile sourced
//	  daemon_service: true                 # optional, default true
//
// On a sprite the nix store binaries live under /nix and nix-installed tools land
// in ~/.nix-profile/bin, neither of which is on the base (PID 1) PATH that
// `sprite exec` and sprite services inherit. The module therefore symlinks the
// `nix` CLI and each declared package binary into ~/.local/bin (which is on that
// PATH) and sources the full nix profile from the login shells for interactive
// use, mirroring how apt/corepack/cargo expose their tools.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

const (
	nixBin           = "/nix/var/nix/profiles/default/bin/nix"
	nixDaemonBin     = "/nix/var/nix/profiles/default/bin/nix-daemon"
	nixDaemonProfile = "/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh"
	nixUserProfile   = "~/.nix-profile/bin"
	nixService       = "nix-daemon"
	nixInstallURL    = "https://install.determinate.systems/nix"
	nixRCBegin       = "# BEGIN SPROOT NIX BLOCK"
	nixRCEnd         = "# END SPROOT NIX BLOCK"
)

// nixRCBody is the static block sourced from ~/.bashrc and ~/.zshrc so interactive
// login shells get the full nix environment (NIX_PATH, NIX_SSL_CERT_FILE, the
// profile bin dirs, etc.). Non-login shells rely on the ~/.local/bin symlinks.
const nixRCBody = `# Load the Nix profile.
if [ -e ` + nixDaemonProfile + ` ]; then
  . ` + nixDaemonProfile + `
fi`

func init() {
	phase.Register("nix", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.Nix == nil {
			return nil, fmt.Errorf("nix: config is nil")
		}
		return &nixPhase{cfg: cfg.Nix}, nil
	})
}

type nixPhase struct {
	cfg *config.NixConfig
}

func (p *nixPhase) Type() string { return "nix" }
func (p *nixPhase) Name() string { return "nix" }

func (p *nixPhase) ShouldRun(_ *phase.Context) (bool, error) {
	// Not installed yet.
	if _, err := os.Stat(nixBin); err != nil {
		return true, nil
	}
	bin, err := userLocalBin()
	if err != nil {
		return true, nil
	}
	// The nix CLI must be linked onto the base PATH.
	if _, err := os.Lstat(filepath.Join(bin, "nix")); err != nil {
		return true, nil
	}
	// Each declared package must be linked onto the base PATH.
	for _, pkg := range p.cfg.Packages {
		if _, err := os.Lstat(filepath.Join(bin, pkg.BinName())); err != nil {
			return true, nil
		}
	}
	// The login-shell profile block must be present.
	present, err := nixRCPresent()
	if err != nil || !present {
		return true, nil
	}
	return false, nil
}

func (p *nixPhase) Run(ctx *phase.Context) error {
	if err := p.install(ctx); err != nil {
		return err
	}
	if err := p.ensureDaemon(ctx); err != nil {
		return err
	}
	if err := p.installPackages(ctx); err != nil {
		return err
	}
	if err := p.linkBinaries(ctx); err != nil {
		return err
	}
	if err := p.writeRCBlock(ctx); err != nil {
		return err
	}
	return p.runSetupScript(ctx)
}

// install runs the Determinate Nix installer with no init system (sprites have no
// systemd; nix-daemon is run as a sprite service instead). The installer
// self-escalates via sudo. Skipped when nix is already present.
func (p *nixPhase) install(ctx *phase.Context) error {
	if _, err := os.Stat(nixBin); err == nil {
		ctx.Log.Infof("nix already installed at %s", nixBin)
		return nil
	}
	ctx.Log.Infof("installing Determinate Nix (init=none)")
	cmd := fmt.Sprintf(
		"curl --proto '=https' --tlsv1.2 -sSf -L %s | sh -s -- install linux --init none --no-confirm",
		nixInstallURL,
	)
	if err := runCmd(ctx.Log, "sh", "-c", cmd); err != nil {
		return fmt.Errorf("nix: install: %w", err)
	}
	return nil
}

// ensureDaemon makes sure nix-daemon is running so package installs can talk to
// the store. On a sprite it registers nix-daemon as a managed service (which both
// starts it now and restarts it on boot); elsewhere it starts a background daemon.
// The socket file appears before the daemon accepts connections, so readiness is
// polled with an actual `nix store info` call.
func (p *nixPhase) ensureDaemon(ctx *phase.Context) error {
	if nixDaemonReady() {
		ctx.Log.Infof("nix-daemon already responding")
		return nil
	}
	switch {
	case spriteEnvPresent() && p.cfg.DaemonServiceEnabled():
		// nix-daemon must run as root to write the store and its db. Sprite
		// services run as the unprivileged sprite user, so wrap the daemon in
		// sudo -n (sprites grant passwordless sudo).
		body, err := json.Marshal(map[string]any{
			"cmd":  "sudo",
			"args": []string{"-n", nixDaemonBin},
		})
		if err != nil {
			return fmt.Errorf("nix: marshal service body: %w", err)
		}
		ctx.Log.Infof("registering %s as a sprite service", nixService)
		if err := runCmd(ctx.Log, "sprite-env", "curl",
			"-X", "PUT", "/v1/services/"+nixService, "-d", string(body)); err != nil {
			return fmt.Errorf("nix: register daemon service: %w", err)
		}
	default:
		// No sprite-env (a plain host or CI): start a background daemon. It needs
		// root, so escalate via sudo unless we already are root.
		ctx.Log.Infof("starting nix-daemon in the background")
		inner := fmt.Sprintf("nohup %s >/tmp/nix-daemon.log 2>&1 &", nixDaemonBin)
		start := inner
		if os.Geteuid() != 0 {
			start = "sudo -n sh -c " + shellQuote(inner)
		}
		if err := runCmd(ctx.Log, "sh", "-c", start); err != nil {
			return fmt.Errorf("nix: start daemon: %w", err)
		}
	}
	if !waitForDaemon(20) {
		return fmt.Errorf("nix: daemon did not become ready")
	}
	return nil
}

// installPackages installs each declared package into the user profile. A package
// already present in the profile (its binary exists under ~/.nix-profile/bin) is
// skipped so re-runs are cheap.
func (p *nixPhase) installPackages(ctx *phase.Context) error {
	profileBin, err := expandTilde(nixUserProfile)
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	for _, pkg := range p.cfg.Packages {
		if _, err := os.Stat(filepath.Join(profileBin, pkg.BinName())); err == nil {
			ctx.Log.Infof("nix package %q already installed", pkg.FlakeRef())
			continue
		}
		ctx.Log.Infof("nix profile install %s", pkg.FlakeRef())
		if err := nixRun(ctx, "nix profile install "+shellQuote(pkg.FlakeRef())); err != nil {
			return fmt.Errorf("nix: install %q: %w", pkg.FlakeRef(), err)
		}
	}
	return nil
}

// linkBinaries symlinks the nix CLI and each declared package binary into
// ~/.local/bin, which is on the base PATH inherited by sprite exec and services.
func (p *nixPhase) linkBinaries(ctx *phase.Context) error {
	bin, err := userLocalBin()
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	if err := runCmd(ctx.Log, "ln", "-sf", nixBin, filepath.Join(bin, "nix")); err != nil {
		return fmt.Errorf("nix: link nix cli: %w", err)
	}
	profileBin, err := expandTilde(nixUserProfile)
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	for _, pkg := range p.cfg.Packages {
		name := pkg.BinName()
		from := filepath.Join(profileBin, name)
		to := filepath.Join(bin, name)
		if err := runCmd(ctx.Log, "ln", "-sf", from, to); err != nil {
			return fmt.Errorf("nix: link %q: %w", name, err)
		}
	}
	return nil
}

func (p *nixPhase) writeRCBlock(ctx *phase.Context) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	for _, rc := range []string{".bashrc", ".zshrc"} {
		path := filepath.Join(home, rc)
		if err := applyManagedBlock(path, nixRCBody, nixRCBegin, nixRCEnd); err != nil {
			return fmt.Errorf("nix: %s: %w", rc, err)
		}
		ctx.Log.Infof("wrote nix profile block to %s", path)
	}
	return nil
}

// runSetupScript runs the optional user setup script with the nix profile sourced.
// The script path is resolved relative to the config repo, like file_template src.
func (p *nixPhase) runSetupScript(ctx *phase.Context) error {
	if p.cfg.SetupScript == "" {
		return nil
	}
	scriptPath := filepath.Join(ctx.ConfigRepoPath, p.cfg.SetupScript)
	if _, err := os.Stat(scriptPath); err != nil {
		return fmt.Errorf("nix: setup_script %q: %w", p.cfg.SetupScript, err)
	}
	ctx.Log.Infof("running nix setup script %s", p.cfg.SetupScript)
	return nixRun(ctx, "sh "+shellQuote(scriptPath))
}

func (p *nixPhase) Verify(_ *phase.Context) error {
	bin, err := userLocalBin()
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	if _, err := os.Lstat(filepath.Join(bin, "nix")); err != nil {
		return fmt.Errorf("nix: nix cli not linked into %s", bin)
	}
	for _, pkg := range p.cfg.Packages {
		if _, err := os.Lstat(filepath.Join(bin, pkg.BinName())); err != nil {
			return fmt.Errorf("nix: %q not linked into %s after install", pkg.BinName(), bin)
		}
	}
	present, err := nixRCPresent()
	if err != nil {
		return fmt.Errorf("nix: %w", err)
	}
	if !present {
		return fmt.Errorf("nix: profile block missing from rc files")
	}
	return nil
}

// nixRun runs a shell snippet with the nix daemon profile sourced first, so PATH,
// NIX_PATH and the SSL cert env are set for nix subcommands.
func nixRun(ctx *phase.Context, script string) error {
	return runCmd(ctx.Log, "sh", "-c", ". "+nixDaemonProfile+" >/dev/null 2>&1; "+script)
}

// nixDaemonReady reports whether the nix store daemon answers a trivial query.
func nixDaemonReady() bool {
	return checkCmd("sh", "-c", ". "+nixDaemonProfile+" >/dev/null 2>&1; nix store info >/dev/null 2>&1")
}

// waitForDaemon polls nixDaemonReady up to attempts times (one second apart).
func waitForDaemon(attempts int) bool {
	for i := 0; i < attempts; i++ {
		if nixDaemonReady() {
			return true
		}
		time.Sleep(time.Second)
	}
	return false
}

// nixRCPresent reports whether the managed nix block is present in both rc files.
func nixRCPresent() (bool, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	want := blockHash(nixRCBody + "\n")
	for _, rc := range []string{".bashrc", ".zshrc"} {
		content, err := os.ReadFile(filepath.Join(home, rc))
		if err != nil {
			return false, nil
		}
		if extractManagedBlockHash(string(content), nixRCBegin, nixRCEnd) != want {
			return false, nil
		}
	}
	return true, nil
}

// shellQuote wraps s in single quotes for safe inclusion in a sh -c string,
// escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
