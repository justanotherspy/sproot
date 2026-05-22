package sprite

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	_ "github.com/justanotherspy/sproot/internal/phase/modules"
	"github.com/justanotherspy/sproot/pkg/log"
)

// SetupOptions controls the behavior of RunSetup.
type SetupOptions struct {
	ConfigRepo  string
	Ref         string
	ConfigPath  string // path to config file within the cloned repo; defaults to "sproot.yaml"
	Target      string // named target from sproot.yaml; empty uses default/flat phases
	LocalConfig string // absolute path to config directory already in the sprite; skips git clone
	Only        string
	Force       bool
	DryRun      bool
	Status      bool
	SkipVerify  bool // skip the built-in verify phase (useful for minimal/partial configs)
}

const defaultConfigRepoDir = "~/.sproot/config-repo"

// RunSetup is the entry point for the in-sprite setup command. It clones (or
// updates) the config repo, loads sproot.yaml, builds phases, appends the
// built-in verify phase, and runs everything via the phase runner.
func RunSetup(opts SetupOptions) error {
	l := log.Stderr()

	if opts.Status {
		statePath, err := phase.DefaultStatePath()
		if err != nil {
			return fmt.Errorf("state path: %w", err)
		}
		return PrintStatus(statePath, os.Stdout)
	}

	var destDir string
	if opts.LocalConfig != "" {
		destDir = opts.LocalConfig
	} else {
		if opts.ConfigRepo == "" {
			return fmt.Errorf("--config-repo is required")
		}
		var err error
		destDir, err = config.ExpandTilde(defaultConfigRepoDir)
		if err != nil {
			return fmt.Errorf("expand config repo dir: %w", err)
		}
		if err := cloneOrPull(l, opts.ConfigRepo, opts.Ref, destDir); err != nil {
			return fmt.Errorf("clone config repo: %w", err)
		}
	}

	configFile := opts.ConfigPath
	if configFile == "" {
		configFile = "sproot.yaml"
	}
	l.Debugf("loading config from %s", filepath.Join(destDir, configFile))
	cfg, err := config.LoadSprootConfig(filepath.Join(destDir, configFile))
	if err != nil {
		return fmt.Errorf("load %s: %w", configFile, err)
	}

	if err := config.ValidateSprootConfig(cfg); err != nil {
		return fmt.Errorf("invalid %s: %w", configFile, err)
	}

	resolvedPhases, err := cfg.ResolveTarget(opts.Target)
	if err != nil {
		return fmt.Errorf("resolve target: %w", err)
	}
	l.Debugf("resolved %d phase(s) for target %q", len(resolvedPhases), opts.Target)

	phases := make([]phase.Phase, 0, len(resolvedPhases)+1)
	for _, phaseCfg := range resolvedPhases {
		p, err := phase.Build(phaseCfg)
		if err != nil {
			return fmt.Errorf("build phase %q: %w", phaseCfg.Type, err)
		}
		phases = append(phases, p)
	}
	if !opts.SkipVerify {
		phases = append(phases, newVerifyPhase())
	}

	runner := phase.NewRunnerFromPhases(phases, phase.RunnerOptions{Only: opts.Only})
	ctx := &phase.Context{
		ConfigRepoPath: destDir,
		Identity:       cfg.Identity,
		Log:            l,
		DryRun:         opts.DryRun,
		Force:          opts.Force,
		OnlyFilter:     opts.Only,
	}

	return runner.Run(ctx)
}

// cloneOrPull clones repoURL at ref into dest. If dest already contains a git
// repository it fetches and checks out the requested ref instead.
// If the recorded remote URL differs from repoURL, the remote is updated first.
func cloneOrPull(l *log.Logger, repoURL, ref, dest string) error {
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		l.Debugf("existing repo found at %s; will fetch and checkout", dest)
		if current, err := gitOutput("-C", dest, "remote", "get-url", "origin"); err == nil {
			if strings.TrimSpace(current) != repoURL {
				l.Infof("config_repo URL changed; updating remote to %s", repoURL)
				if err := runGit("-C", dest, "remote", "set-url", "origin", repoURL); err != nil {
					return fmt.Errorf("update remote URL: %w", err)
				}
			}
		}
		l.Infof("updating config repo at %s", dest)
		if err := runGit("-C", dest, "fetch", "--prune", "origin"); err != nil {
			return err
		}
		l.Debugf("checking out ref %s", ref)
		if err := runGit("-C", dest, "checkout", ref); err != nil {
			return err
		}
		// Fast-forward pull; silently ignored for detached HEAD (SHA or tag).
		_ = runGit("-C", dest, "merge", "--ff-only")
		return nil
	}
	l.Infof("cloning %s (ref %s)", repoURL, ref)
	return runGit("clone", "--branch", ref, repoURL, dest)
}

// runGit runs git with the given arguments, streaming stdout and stderr directly
// to the process's own stdout/stderr so progress is visible.
func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// gitOutput runs git and returns its trimmed stdout.
func gitOutput(args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.Command("git", args...)
	cmd.Stdout = &buf
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return buf.String(), nil
}
