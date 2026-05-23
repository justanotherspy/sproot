package host

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/pkg/log"
)

func TestParseLsRemote(t *testing.T) {
	tests := []struct {
		name string
		out  string
		ref  string
		want string
	}{
		{
			name: "branch",
			out:  "abc123\trefs/heads/main\ndef456\trefs/heads/dev\n",
			ref:  "main",
			want: "abc123",
		},
		{
			name: "lightweight tag",
			out:  "aaa111\trefs/tags/v1.0.0\n",
			ref:  "v1.0.0",
			want: "aaa111",
		},
		{
			name: "annotated tag prefers dereferenced commit",
			out:  "tagobj0\trefs/tags/v1.0.0\ncommit0\trefs/tags/v1.0.0^{}\n",
			ref:  "v1.0.0",
			want: "commit0",
		},
		{
			name: "branch preferred over same-named tag",
			out:  "tagsha0\trefs/tags/rel\nheadsha\trefs/heads/rel\n",
			ref:  "rel",
			want: "headsha",
		},
		{
			name: "fallback to first line",
			out:  "weird0\trefs/something/odd\n",
			ref:  "main",
			want: "weird0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseLsRemote(tt.out, tt.ref)
			if err != nil {
				t.Fatalf("parseLsRemote: %v", err)
			}
			if got != tt.want {
				t.Errorf("parseLsRemote = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParseLsRemote_NotFound(t *testing.T) {
	if _, err := parseLsRemote("", "main"); err == nil {
		t.Fatal("expected error for empty ls-remote output")
	}
}

func TestConfigCacheRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	c := configCache{Entries: map[string]configCacheEntry{
		configCacheKey("repo", "main", "sproot.yaml"): {CommitSHA: "deadbeef", RawConfig: "schema_version: 1\n"},
	}}
	writeConfigCache(c)

	got := readConfigCache()
	entry, ok := got.Entries[configCacheKey("repo", "main", "sproot.yaml")]
	if !ok {
		t.Fatal("entry not found after round trip")
	}
	if entry.CommitSHA != "deadbeef" || entry.RawConfig != "schema_version: 1\n" {
		t.Errorf("round trip mismatch: %+v", entry)
	}
}

func TestReadConfigCache_MissingIsEmpty(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	c := readConfigCache()
	if c.Entries == nil {
		t.Fatal("Entries should be a non-nil empty map")
	}
	if len(c.Entries) != 0 {
		t.Fatalf("expected empty cache, got %d entries", len(c.Entries))
	}
}

// initGitRepo creates a git repo at dir with a sproot.yaml holding content, on
// branch main, and returns the HEAD commit SHA.
func initGitRepo(t *testing.T, dir, content string) string {
	t.Helper()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")
	if err := writeFile(filepath.Join(dir, "sproot.yaml"), content); err != nil {
		t.Fatal(err)
	}
	run("add", "sproot.yaml")
	run("commit", "-m", "init")
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("rev-parse: %v", err)
	}
	return string(out[:len(out)-1]) // trim trailing newline
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}

func TestLoadConfigBytes_UsesCacheWhenRefUnchanged(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	initGitRepo(t, repo, "schema_version: 1\n# original\n")
	l := log.New(io.Discard)

	// First load populates the cache from a real clone.
	raw, err := loadConfigBytes(repo, "main", "sproot.yaml", l)
	if err != nil {
		t.Fatalf("first loadConfigBytes: %v", err)
	}
	if string(raw) != "schema_version: 1\n# original\n" {
		t.Fatalf("unexpected content: %q", raw)
	}

	// Overwrite the cached content (keeping the commit SHA) with a sentinel.
	// A second load with the ref unchanged must return the sentinel, proving it
	// served from cache rather than re-cloning.
	cache := readConfigCache()
	key := configCacheKey(repo, "main", "sproot.yaml")
	entry := cache.Entries[key]
	entry.RawConfig = "SENTINEL FROM CACHE\n"
	cache.Entries[key] = entry
	writeConfigCache(cache)

	raw2, err := loadConfigBytes(repo, "main", "sproot.yaml", l)
	if err != nil {
		t.Fatalf("second loadConfigBytes: %v", err)
	}
	if string(raw2) != "SENTINEL FROM CACHE\n" {
		t.Errorf("expected cached sentinel, got %q (cache was not used)", raw2)
	}
}

func TestLoadConfigBytes_RefetchesWhenRefChanges(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	t.Setenv("HOME", t.TempDir())
	repo := t.TempDir()
	initGitRepo(t, repo, "schema_version: 1\n# v1\n")
	l := log.New(io.Discard)

	if _, err := loadConfigBytes(repo, "main", "sproot.yaml", l); err != nil {
		t.Fatalf("first loadConfigBytes: %v", err)
	}

	// Commit a new sproot.yaml; the ref now points at a different commit, so the
	// cache must be invalidated and the new content returned.
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	if err := writeFile(filepath.Join(repo, "sproot.yaml"), "schema_version: 1\n# v2\n"); err != nil {
		t.Fatal(err)
	}
	run("add", "sproot.yaml")
	run("commit", "-m", "update")

	raw, err := loadConfigBytes(repo, "main", "sproot.yaml", l)
	if err != nil {
		t.Fatalf("second loadConfigBytes: %v", err)
	}
	if string(raw) != "schema_version: 1\n# v2\n" {
		t.Errorf("expected refreshed v2 content, got %q", raw)
	}
}

func TestEffectiveCloneURL(t *testing.T) {
	const ssh = "git@github.com:owner/repo.git"
	const https = "https://github.com/owner/repo.git"

	t.Run("non-github url returned unchanged without probing", func(t *testing.T) {
		called := false
		swapLsRemote(t, func(string, string) (string, error) {
			called = true
			return "", nil
		})
		if got := effectiveCloneURL("file:///tmp/repo", "main", log.New(io.Discard)); got != "file:///tmp/repo" {
			t.Errorf("got %q; want unchanged", got)
		}
		if called {
			t.Error("ls-remote should not be probed for a non-github-ssh URL")
		}
	})

	t.Run("keeps ssh when ssh works", func(t *testing.T) {
		swapLsRemote(t, func(repo, _ string) (string, error) {
			if repo != ssh {
				t.Errorf("expected ssh probe first, got %q", repo)
			}
			return "sha", nil
		})
		if got := effectiveCloneURL(ssh, "main", log.New(io.Discard)); got != ssh {
			t.Errorf("got %q; want ssh kept (private-repo access preserved)", got)
		}
	})

	t.Run("falls back to https when ssh fails", func(t *testing.T) {
		swapLsRemote(t, func(repo, _ string) (string, error) {
			if repo == ssh {
				return "", errProbe
			}
			return "sha", nil
		})
		if got := effectiveCloneURL(ssh, "main", log.New(io.Discard)); got != https {
			t.Errorf("got %q; want https fallback", got)
		}
	})

	t.Run("keeps ssh when both fail so the error names the configured URL", func(t *testing.T) {
		swapLsRemote(t, func(string, string) (string, error) { return "", errProbe })
		if got := effectiveCloneURL(ssh, "main", log.New(io.Discard)); got != ssh {
			t.Errorf("got %q; want original ssh URL", got)
		}
	})
}

var errProbe = errProbeT("ls-remote failed")

type errProbeT string

func (e errProbeT) Error() string { return string(e) }

// swapLsRemote replaces lsRemoteFn for the duration of the test.
func swapLsRemote(t *testing.T, fn func(string, string) (string, error)) {
	t.Helper()
	orig := lsRemoteFn
	lsRemoteFn = fn
	t.Cleanup(func() { lsRemoteFn = orig })
}
