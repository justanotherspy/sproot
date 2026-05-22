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
		Repos:   []config.RepoCloneEntry{{Raw: "owner/myrepo"}},
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
	repoDir := filepath.Join(base, "myrepo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755)

	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []config.RepoCloneEntry{{Raw: "owner/myrepo"}},
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
	repoDir := filepath.Join(base, "myrepo")
	_ = os.MkdirAll(filepath.Join(repoDir, ".git"), 0o755)

	ctx := testCtx(t)
	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []config.RepoCloneEntry{{Raw: "owner/myrepo"}},
	}}
	if err := p.Run(ctx); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !isGitRepo(repoDir) {
		t.Error("repo dir lost after Run")
	}
}

func TestRepoClone_URLFormWithDest(t *testing.T) {
	dest := t.TempDir()
	_ = os.MkdirAll(filepath.Join(dest, ".git"), 0o755)

	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		Repos: []config.RepoCloneEntry{{URL: "https://github.com/org/project.git", Dest: dest}},
	}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when dest is already a git repo")
	}
}

func TestRepoClone_URLFormWithDestMissing(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "nonexistent")

	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		Repos: []config.RepoCloneEntry{{URL: "https://github.com/org/project.git", Dest: dest}},
	}}

	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true when dest is missing")
	}
}

func TestRepoClone_VerifyFailsWhenMissing(t *testing.T) {
	base := t.TempDir()

	p := &repoClonePhase{cfg: &config.RepoCloneConfig{
		BaseDir: base,
		Repos:   []config.RepoCloneEntry{{Raw: "owner/missingrepo"}},
	}}

	if err := p.Verify(testCtx(t)); err == nil {
		t.Error("expected Verify to fail when repo not cloned")
	}
}

func TestEntryDest_ShortForm(t *testing.T) {
	base := t.TempDir()
	entry := config.RepoCloneEntry{Raw: "owner/myrepo"}
	dest, err := entryDest(entry, base)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(base, "myrepo")
	if dest != want {
		t.Errorf("got %q, want %q", dest, want)
	}
}

func TestEntryDest_URLFormWithDest(t *testing.T) {
	explicitDest := "/tmp/myproject"
	entry := config.RepoCloneEntry{URL: "https://github.com/org/project.git", Dest: explicitDest}
	dest, err := entryDest(entry, "")
	if err != nil {
		t.Fatal(err)
	}
	if dest != explicitDest {
		t.Errorf("got %q, want %q", dest, explicitDest)
	}
}

func TestEntryCloneURL_ShortForm(t *testing.T) {
	entry := config.RepoCloneEntry{Raw: "owner/repo"}
	got := entryCloneURL(entry)
	want := "git@github.com:owner/repo.git"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestEntryCloneURL_URLForm(t *testing.T) {
	url := "https://github.com/org/project.git"
	entry := config.RepoCloneEntry{URL: url}
	got := entryCloneURL(entry)
	if got != url {
		t.Errorf("got %q, want %q", got, url)
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
