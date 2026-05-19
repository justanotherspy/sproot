package modules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestRepoClone_ShouldRunWhenDirMissing(t *testing.T) {
	base := t.TempDir()
	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []string{"owner/myrepo"},
	}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when target dir missing")
	}
}

func TestRepoClone_ShouldRunFalseWhenAllPresent(t *testing.T) {
	base := t.TempDir()
	// Create a fake git repo dir.
	repoDir := filepath.Join(base, "myrepo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755)

	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []string{"owner/myrepo"},
	}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when target dir already a git repo")
	}
}

func TestRepoClone_SkipsAlreadyCloned(t *testing.T) {
	base := t.TempDir()
	// Pre-populate a fake git repo to simulate already-cloned.
	repoDir := filepath.Join(base, "myrepo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755)

	ctx := testCtx(t)
	logged := false
	// Run should skip without calling git clone.
	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []string{"owner/myrepo"},
	}}
	_ = logged
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	// The dir still exists and no git error occurred.
	if !isGitRepo(repoDir) {
		t.Error("repo dir lost after Run")
	}
}

func TestSplitRepo(t *testing.T) {
	cases := []struct {
		in, wantOwner, wantName string
	}{
		{"owner/repo", "owner", "repo"},
		{"justanotherspy/garlic", "justanotherspy", "garlic"},
		{"noslash", "", "noslash"},
	}
	for _, tc := range cases {
		owner, name := splitRepo(tc.in)
		if owner != tc.wantOwner || name != tc.wantName {
			t.Errorf("splitRepo(%q): got (%q,%q), want (%q,%q)",
				tc.in, owner, name, tc.wantOwner, tc.wantName)
		}
	}
}

func TestIsGitRepo(t *testing.T) {
	dir := t.TempDir()
	if isGitRepo(dir) {
		t.Error("plain dir should not be a git repo")
	}
	_ = os.MkdirAll(filepath.Join(dir, ".git"), 0o755)
	if !isGitRepo(dir) {
		t.Error("dir with .git should be a git repo")
	}
}
