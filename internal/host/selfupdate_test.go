package host

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIsNewer(t *testing.T) {
	cases := []struct {
		latest, current string
		want            bool
	}{
		{"v1.2.0", "v1.1.0", true},
		{"1.2.0", "1.1.9", true},
		{"v1.2.0", "1.2.0", false},
		{"v1.1.0", "v1.2.0", false},
		{"v1.2.0", "dev", false}, // dev never compares as older
		{"not-a-version", "v1.0.0", false},
		{"v2.0.0", "v1.9.9", true},
	}
	for _, c := range cases {
		if got := isNewer(c.latest, c.current); got != c.want {
			t.Errorf("isNewer(%q, %q) = %v, want %v", c.latest, c.current, got, c.want)
		}
	}
}

func TestReleaseArchiveName(t *testing.T) {
	cases := []struct {
		ver, goos, goarch, want string
	}{
		{"0.1.0", "linux", "amd64", "sproot_0.1.0_linux_amd64.tar.gz"},
		{"0.1.0", "darwin", "arm64", "sproot_0.1.0_darwin_arm64.tar.gz"},
		{"0.1.0", "windows", "amd64", "sproot_0.1.0_windows_amd64.zip"},
	}
	for _, c := range cases {
		if got := releaseArchiveName(c.ver, c.goos, c.goarch); got != c.want {
			t.Errorf("releaseArchiveName(%q,%q,%q) = %q, want %q", c.ver, c.goos, c.goarch, got, c.want)
		}
	}
}

func TestVerifyChecksum(t *testing.T) {
	archive := []byte("pretend archive bytes")
	sum := sha256.Sum256(archive)
	hexSum := hex.EncodeToString(sum[:])
	name := "sproot_0.1.0_linux_amd64.tar.gz"
	checksums := fmt.Sprintf("deadbeef  sproot_0.1.0_darwin_arm64.tar.gz\n%s  %s\n", hexSum, name)

	if err := verifyChecksum(archive, []byte(checksums), name); err != nil {
		t.Fatalf("verifyChecksum on valid input: %v", err)
	}

	if err := verifyChecksum([]byte("tampered"), []byte(checksums), name); err == nil {
		t.Error("expected mismatch error for tampered archive")
	} else if !strings.Contains(err.Error(), "mismatch") {
		t.Errorf("error %q does not mention mismatch", err)
	}

	if err := verifyChecksum(archive, []byte("abc  some-other-file\n"), name); err == nil {
		t.Error("expected error when checksum entry is missing")
	} else if !strings.Contains(err.Error(), "no checksum entry") {
		t.Errorf("error %q does not mention missing entry", err)
	}
}

func TestExtractBinary_TarGz(t *testing.T) {
	want := []byte("the sproot binary")
	archive := makeTarGz(t, map[string][]byte{"sproot": want})
	got, err := extractBinary(archive.Bytes(), "linux")
	if err != nil {
		t.Fatalf("extractBinary tar.gz: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted bytes mismatch: got %q want %q", got, want)
	}
}

func TestExtractBinary_Zip(t *testing.T) {
	want := []byte("the sproot windows binary")
	archive := makeZip(t, "sproot.exe", want)
	got, err := extractBinary(archive, "windows")
	if err != nil {
		t.Fatalf("extractBinary zip: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("extracted bytes mismatch: got %q want %q", got, want)
	}
}

func TestUpdateCacheRoundTrip(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	if _, ok := readUpdateCache(); ok {
		t.Fatal("expected no cache before writing")
	}

	now := time.Now().Truncate(time.Second)
	writeUpdateCache(updateCache{CheckedAt: now, LatestVersion: "v9.9.9"})

	c, ok := readUpdateCache()
	if !ok {
		t.Fatal("expected cache after writing")
	}
	if c.LatestVersion != "v9.9.9" || !c.CheckedAt.Equal(now) {
		t.Errorf("round trip mismatch: %+v", c)
	}

	clearUpdateCache()
	if _, ok := readUpdateCache(); ok {
		t.Error("expected no cache after clear")
	}
}

func TestCachedLatestVersion_FreshCacheSkipsFetch(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	writeUpdateCache(updateCache{CheckedAt: now.Add(-1 * time.Hour), LatestVersion: "v1.0.0"})

	got, err := cachedLatestVersion(now, func(context.Context) (string, error) {
		t.Fatal("latestFn should not be called when cache is fresh")
		return "", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v1.0.0" {
		t.Errorf("got %q, want v1.0.0", got)
	}
}

func TestCachedLatestVersion_StaleCacheRefetches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	writeUpdateCache(updateCache{CheckedAt: now.Add(-48 * time.Hour), LatestVersion: "v1.0.0"})

	got, err := cachedLatestVersion(now, func(context.Context) (string, error) {
		return "v2.0.0", nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v2.0.0" {
		t.Errorf("got %q, want v2.0.0", got)
	}
	// The refreshed value should now be cached with the new timestamp.
	c, ok := readUpdateCache()
	if !ok || c.LatestVersion != "v2.0.0" || !c.CheckedAt.Equal(now) {
		t.Errorf("cache not refreshed: %+v ok=%v", c, ok)
	}
}

func TestCachedLatestVersion_FetchErrorFallsBackToStale(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	now := time.Now()
	writeUpdateCache(updateCache{CheckedAt: now.Add(-48 * time.Hour), LatestVersion: "v1.0.0"})

	got, err := cachedLatestVersion(now, func(context.Context) (string, error) {
		return "", fmt.Errorf("network down")
	})
	if err != nil {
		t.Fatalf("expected fallback to stale cache, got error: %v", err)
	}
	if got != "v1.0.0" {
		t.Errorf("got %q, want stale v1.0.0", got)
	}
}

func TestUpdateNotice(t *testing.T) {
	t.Run("dev build is silent", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		if got := updateNotice("dev", time.Now(), failingLatestFn(t)); got != nil {
			t.Errorf("expected nil for dev build, got %v", got)
		}
	})

	t.Run("newer available emits notice", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		now := time.Now()
		writeUpdateCache(updateCache{CheckedAt: now, LatestVersion: "v2.0.0"})
		got := updateNotice("v1.0.0", now, failingLatestFn(t))
		if len(got) != 2 {
			t.Fatalf("expected 2 notice lines, got %v", got)
		}
		if !strings.Contains(got[0], "v2.0.0") || !strings.Contains(got[0], "v1.0.0") {
			t.Errorf("notice missing versions: %q", got[0])
		}
		if !strings.Contains(got[1], "self-update") {
			t.Errorf("notice missing upgrade hint: %q", got[1])
		}
	})

	t.Run("up to date is silent", func(t *testing.T) {
		t.Setenv("HOME", t.TempDir())
		now := time.Now()
		writeUpdateCache(updateCache{CheckedAt: now, LatestVersion: "v1.0.0"})
		if got := updateNotice("v1.0.0", now, failingLatestFn(t)); got != nil {
			t.Errorf("expected nil when up to date, got %v", got)
		}
	})
}

func TestRunSelfUpdate_DevBuildErrors(t *testing.T) {
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "dev",
		latestFn:       failingLatestFn(t),
		fetchFn:        failingFetchFn(t),
	})
	if err == nil || !strings.Contains(err.Error(), "dev builds") {
		t.Errorf("expected dev build error, got %v", err)
	}
}

func TestRunSelfUpdate_AlreadyUpToDate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		latestFn:       func(context.Context) (string, error) { return "v1.0.0", nil },
		fetchFn:        failingFetchFn(t), // must not download when already current
		execPath:       "/should/not/be/touched",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSelfUpdate_CheckOnlyDoesNotDownload(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		CheckOnly:      true,
		latestFn:       func(context.Context) (string, error) { return "v2.0.0", nil },
		fetchFn:        failingFetchFn(t), // --check must never download
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// --check refreshes the cache so the daily notifier reflects the check.
	c, ok := readUpdateCache()
	if !ok || c.LatestVersion != "v2.0.0" {
		t.Errorf("expected cache refreshed to v2.0.0, got %+v ok=%v", c, ok)
	}
}

func TestRunSelfUpdate_ReplacesBinaryAndClearsCache(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	// Seed a cache that should be cleared after a successful upgrade.
	writeUpdateCache(updateCache{CheckedAt: time.Now(), LatestVersion: "v2.0.0"})

	exe := filepath.Join(home, "sproot")
	if err := os.WriteFile(exe, []byte("OLD BINARY"), 0o755); err != nil {
		t.Fatal(err)
	}

	newBytes := []byte("NEW BINARY CONTENTS")
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		latestFn:       func(context.Context) (string, error) { return "v2.0.0", nil },
		fetchFn:        func(context.Context, string) ([]byte, error) { return newBytes, nil },
		execPath:       exe,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(exe)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, newBytes) {
		t.Errorf("binary not replaced: got %q", got)
	}
	info, err := os.Stat(exe)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("replaced binary is not executable: mode %v", info.Mode())
	}
	if _, ok := readUpdateCache(); ok {
		t.Error("expected update cache to be cleared after upgrade")
	}
}

func TestIsHomebrewManaged(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"/opt/homebrew/Caskroom/sproot/0.1.0/sproot", true},
		{"/home/linuxbrew/.linuxbrew/Caskroom/sproot/0.1.0/sproot", true},
		{"/usr/local/Cellar/sproot/0.1.0/bin/sproot", true},
		{"/usr/local/bin/sproot", false},
		{"/home/user/.local/bin/sproot", false},
	}
	for _, c := range cases {
		if got := isHomebrewManaged(c.path); got != c.want {
			t.Errorf("isHomebrewManaged(%q) = %v, want %v", c.path, got, c.want)
		}
	}
}

func TestRunSelfUpdate_HomebrewDelegates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	called := false
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		execPath:       "/opt/homebrew/Caskroom/sproot/0.1.0/sproot",
		latestFn:       failingLatestFn(t), // a brew install must not query GitHub
		fetchFn:        failingFetchFn(t),  // ...nor download a release archive
		brewUpgradeFn:  func(context.Context) error { called = true; return nil },
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Error("expected brew upgrade to be invoked for a Homebrew-managed install")
	}
}

func TestRunSelfUpdate_HomebrewCheckOnlyDoesNotDelegate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		CheckOnly:      true,
		execPath:       "/opt/homebrew/Caskroom/sproot/0.1.0/sproot",
		latestFn:       func(context.Context) (string, error) { return "v2.0.0", nil },
		fetchFn:        failingFetchFn(t),
		brewUpgradeFn: func(context.Context) error {
			t.Fatal("brew upgrade must not run for --check")
			return nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunSelfUpdate_DownloadErrorPropagates(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	exe := filepath.Join(t.TempDir(), "sproot")
	if err := os.WriteFile(exe, []byte("OLD"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := RunSelfUpdate(context.Background(), SelfUpdateOptions{
		CurrentVersion: "v1.0.0",
		latestFn:       func(context.Context) (string, error) { return "v2.0.0", nil },
		fetchFn:        func(context.Context, string) ([]byte, error) { return nil, fmt.Errorf("boom") },
		execPath:       exe,
	})
	if err == nil || !strings.Contains(err.Error(), "boom") {
		t.Errorf("expected download error to propagate, got %v", err)
	}
	// The original binary must be left untouched on failure.
	got, _ := os.ReadFile(exe)
	if string(got) != "OLD" {
		t.Errorf("binary should be unchanged on download failure, got %q", got)
	}
}

func TestFetchLatestVersionFrom(t *testing.T) {
	t.Run("success sends User-Agent and parses tag", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("User-Agent") == "" {
				t.Error("expected a User-Agent header (GitHub API 403s without one)")
			}
			_, _ = w.Write([]byte(`{"tag_name":"v1.4.2"}`))
		}))
		defer srv.Close()

		got, err := fetchLatestVersionFrom(context.Background(), srv.Client(), srv.URL)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != "v1.4.2" {
			t.Errorf("got %q, want v1.4.2", got)
		}
	})

	t.Run("rate limit yields a friendly error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusForbidden)
		}))
		defer srv.Close()

		_, err := fetchLatestVersionFrom(context.Background(), srv.Client(), srv.URL)
		if err == nil || !strings.Contains(err.Error(), "rate limit") {
			t.Errorf("expected rate limit error, got %v", err)
		}
	})

	t.Run("empty tag is an error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		_, err := fetchLatestVersionFrom(context.Background(), srv.Client(), srv.URL)
		if err == nil || !strings.Contains(err.Error(), "tag_name") {
			t.Errorf("expected missing tag_name error, got %v", err)
		}
	})
}

// --- helpers ---

func failingLatestFn(t *testing.T) func(context.Context) (string, error) {
	t.Helper()
	return func(context.Context) (string, error) {
		t.Fatal("latestFn should not be called")
		return "", nil
	}
}

func failingFetchFn(t *testing.T) func(context.Context, string) ([]byte, error) {
	t.Helper()
	return func(context.Context, string) ([]byte, error) {
		t.Fatal("fetchFn should not be called")
		return nil, nil
	}
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
