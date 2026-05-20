package sprite

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	_ "github.com/justanotherspy/sproot/internal/phase/modules"
	"github.com/justanotherspy/sproot/pkg/log"
)

// SetupOptions controls the behavior of RunSetup.
type SetupOptions struct {
	ConfigRepo string
	Ref        string
	Only       string
	Force      bool
	DryRun     bool
	Status     bool
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

	if opts.ConfigRepo == "" {
		return fmt.Errorf("--config-repo is required")
	}

	destDir, err := config.ExpandTilde(defaultConfigRepoDir)
	if err != nil {
		return fmt.Errorf("expand config repo dir: %w", err)
	}

	if err := cloneOrPull(l, opts.ConfigRepo, opts.Ref, destDir); err != nil {
		return fmt.Errorf("clone config repo: %w", err)
	}

	cfg, err := config.LoadSprootConfig(filepath.Join(destDir, "sproot.yaml"))
	if err != nil {
		return fmt.Errorf("load sproot.yaml: %w", err)
	}

	if err := config.ValidateSprootConfig(cfg); err != nil {
		return fmt.Errorf("invalid sproot.yaml: %w", err)
	}

	phases := make([]phase.Phase, 0, len(cfg.Phases)+1)
	for _, phaseCfg := range cfg.Phases {
		p, err := phase.Build(phaseCfg)
		if err != nil {
			return fmt.Errorf("build phase %q: %w", phaseCfg.Type, err)
		}
		phases = append(phases, p)
	}
	phases = append(phases, newVerifyPhase())

	runner := phase.NewRunnerFromPhases(phases, phase.RunnerOptions{Only: opts.Only})
	ctx := &phase.Context{
		ConfigRepoPath: destDir,
		Identity:       cfg.Identity,
		Log:            l,
		DryRun:         opts.DryRun,
		Force:          opts.Force,
	}

	return runner.Run(ctx)
}

// cloneOrPull clones repoURL at ref into dest. If dest already contains a git
// repository it fetches and checks out the requested ref instead.
func cloneOrPull(l *log.Logger, repoURL, ref, dest string) error {
	if _, err := os.Stat(filepath.Join(dest, ".git")); err == nil {
		l.Infof("updating config repo at %s", dest)
		if err := runGit("-C", dest, "fetch", "--prune", "origin"); err != nil {
			return err
		}
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
