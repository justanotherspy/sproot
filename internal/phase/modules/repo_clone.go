package modules

// repo_clone clones repos into configured destinations.
// Short form: "owner/repo" — cloned via SSH into base_dir/<repo>.
// Long form: {url: "https://...", dest: "~/dir"} — dest is optional.
//
//	- type: repo_clone
//	  base_dir: ~/repos
//	  repos:
//	    - justanotherspy/garlic
//	    - url: https://github.com/org/project.git
//	      dest: ~/project

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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

func (p *repoClonePhase) ShouldRun(_ *phase.Context) (bool, error) {
	for _, entry := range p.cfg.Repos {
		dest, err := entryDest(entry, p.cfg.BaseDir)
		if err != nil {
			return true, nil
		}
		if !isGitRepo(dest) {
			return true, nil
		}
	}
	return false, nil
}

func (p *repoClonePhase) Run(ctx *phase.Context) error {
	for _, entry := range p.cfg.Repos {
		dest, err := entryDest(entry, p.cfg.BaseDir)
		if err != nil {
			return err
		}
		if isGitRepo(dest) {
			ctx.Log.Infof("skipping %s (already cloned)", entryLabel(entry))
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("repo_clone: mkdir %s: %w", filepath.Dir(dest), err)
		}
		cloneURL := entryCloneURL(entry)
		if err := runCmd(ctx.Log, "git", "clone", cloneURL, dest); err != nil {
			return fmt.Errorf("repo_clone: clone %s: %w", entryLabel(entry), err)
		}
	}
	return nil
}

func (p *repoClonePhase) Verify(_ *phase.Context) error {
	for _, entry := range p.cfg.Repos {
		dest, err := entryDest(entry, p.cfg.BaseDir)
		if err != nil {
			return err
		}
		if !isGitRepo(dest) {
			return fmt.Errorf("repo_clone: %s was not cloned to %s", entryLabel(entry), dest)
		}
	}
	return nil
}

// entryCloneURL returns the git URL to clone for an entry.
func entryCloneURL(e config.RepoCloneEntry) string {
	if e.Raw != "" {
		return "git@github.com:" + e.Raw + ".git"
	}
	return e.URL
}

// entryDest returns the resolved destination directory for an entry.
func entryDest(e config.RepoCloneEntry, baseDir string) (string, error) {
	if e.Raw != "" {
		_, name := splitRepo(e.Raw)
		base, err := expandTilde(baseDir)
		if err != nil {
			return "", err
		}
		return filepath.Join(base, name), nil
	}
	if e.Dest != "" {
		return expandTilde(e.Dest)
	}
	name := strings.TrimSuffix(path.Base(e.URL), ".git")
	return expandTilde("~/" + name)
}

// entryLabel returns a short human-readable label for log messages.
func entryLabel(e config.RepoCloneEntry) string {
	if e.Raw != "" {
		return e.Raw
	}
	return e.URL
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
