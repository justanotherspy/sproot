package modules

// repo_clone clones a list of GitHub repos via SSH into a base directory.
// Repos are specified as "owner/repo". Already-cloned directories are skipped.
//
//	- type: repo_clone
//	  base_dir: ~/repos
//	  repos:
//	    - justanotherspy/garlic
//	    - justanotherspy/poker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
)

func init() {
	phase.Register("repo_clone", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.RepoClone == nil {
			return nil, fmt.Errorf("repo_clone: config is nil")
		}
		return &repoClonePhase{cfg: cfg.RepoClone}, nil
	})
}

type repoClonePhase struct {
	cfg *config.RepoCloneConfig
}

func (p *repoClonePhase) Type() string { return "repo_clone" }
func (p *repoClonePhase) Name() string { return "repo_clone" }

func (p *repoClonePhase) ShouldRun(ctx *phase.Context) (bool, error) {
	base, err := expandTilde(p.cfg.BaseDir)
	if err != nil {
		return true, nil
	}
	for _, repo := range p.cfg.Repos {
		_, name := splitRepo(repo)
		dest := filepath.Join(base, name)
		if !isGitRepo(dest) {
			return true, nil
		}
	}
	return false, nil
}

func (p *repoClonePhase) Run(ctx *phase.Context) error {
	base, err := expandTilde(p.cfg.BaseDir)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(base, 0o755); err != nil {
		return fmt.Errorf("repo_clone: mkdir base: %w", err)
	}
	for _, repo := range p.cfg.Repos {
		_, name := splitRepo(repo)
		dest := filepath.Join(base, name)
		if isGitRepo(dest) {
			ctx.Log.Infof("skipping %s (already cloned)", repo)
			continue
		}
		url := "git@github.com:" + repo + ".git"
		if err := runCmd(ctx.Log, "git", "clone", url, dest); err != nil {
			return fmt.Errorf("repo_clone: clone %s: %w", repo, err)
		}
	}
	return nil
}

func (p *repoClonePhase) Verify(_ *phase.Context) error {
	base, err := expandTilde(p.cfg.BaseDir)
	if err != nil {
		return err
	}
	for _, repo := range p.cfg.Repos {
		_, name := splitRepo(repo)
		dest := filepath.Join(base, name)
		if !isGitRepo(dest) {
			return fmt.Errorf("repo_clone: %s was not cloned to %s", repo, dest)
		}
	}
	return nil
}

// splitRepo splits "owner/repo" into ("owner", "repo").
func splitRepo(repo string) (string, string) {
	for i, c := range repo {
		if c == '/' {
			return repo[:i], repo[i+1:]
		}
	}
	return "", repo
}

// isGitRepo returns true if path exists and contains a .git entry.
func isGitRepo(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && info != nil
}
