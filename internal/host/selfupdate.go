package host

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"

	"github.com/justanotherspy/sproot/pkg/log"
)

const (
	latestReleaseAPI    = "https://api.github.com/repos/justanotherspy/sproot/releases/latest"
	releaseDownloadBase = "https://github.com/justanotherspy/sproot/releases/download"
	updateCheckInterval = 24 * time.Hour
	noUpdateCheckEnv    = "SPROOT_NO_UPDATE_CHECK"
)

// updateCheckClient bounds the daily background version check so it can never
// noticeably slow a command. The explicit self-update download reuses
// binaryDownloadClient (a longer timeout) instead.
var updateCheckClient = &http.Client{Timeout: 3 * time.Second}

// updateCache is the on-disk record of the last successful version check,
// stored at ~/.sproot/update-check.json.
type updateCache struct {
	CheckedAt     time.Time `json:"checked_at"`
	LatestVersion string    `json:"latest_version"`
}

func updateCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sproot", "update-check.json"), nil
}

func readUpdateCache() (updateCache, bool) {
	path, err := updateCachePath()
	if err != nil {
		return updateCache{}, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return updateCache{}, false
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return updateCache{}, false
	}
	return c, true
}

// writeUpdateCache persists c atomically (temp file plus rename) so concurrent
// sproot invocations cannot observe a half-written cache. Errors are ignored:
// the cache is an optimization, not a source of truth.
func writeUpdateCache(c updateCache) {
	path, err := updateCachePath()
	if err != nil {
		return
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return
	}
	data, err := json.Marshal(c)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".update-check-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(tmpName)
	}
}

// clearUpdateCache removes the cached check so the next command re-queries upstream.
func clearUpdateCache() {
	if path, err := updateCachePath(); err == nil {
		_ = os.Remove(path)
	}
}

// fetchLatestVersion returns the tag_name of the latest published GitHub release
// (drafts and prereleases excluded by the API). A User-Agent is required or the
// GitHub API rejects the request with HTTP 403.
func fetchLatestVersion(ctx context.Context, client *http.Client) (string, error) {
	return fetchLatestVersionFrom(ctx, client, latestReleaseAPI)
}

func fetchLatestVersionFrom(ctx context.Context, client *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "sproot")
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusForbidden && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		return "", fmt.Errorf("github API rate limit exceeded; try again later")
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("github releases API returned HTTP %d", resp.StatusCode)
	}
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", fmt.Errorf("decode github release response: %w", err)
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("github release response had no tag_name")
	}
	return rel.TagName, nil
}

// isNewer reports whether latest is a strictly higher semantic version than
// current. Either value may carry a leading "v". Unparseable versions (such as
// the "dev" placeholder) yield false so the user is never nagged on bad data.
func isNewer(latest, current string) bool {
	lv, err := semver.NewVersion(strings.TrimSpace(latest))
	if err != nil {
		return false
	}
	cv, err := semver.NewVersion(strings.TrimSpace(current))
	if err != nil {
		return false
	}
	return lv.GreaterThan(cv)
}

// displayVersion normalizes a version to a single "vX.Y.Z" form for messages.
func displayVersion(v string) string {
	return "v" + strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// cachedLatestVersion returns the latest known release version, serving a cached
// value younger than updateCheckInterval and otherwise querying upstream and
// refreshing the cache. On a fetch error it falls back to any cached value.
// now and latestFn are injectable for tests.
func cachedLatestVersion(now time.Time, latestFn func(context.Context) (string, error)) (string, error) {
	if cached, ok := readUpdateCache(); ok {
		if now.Sub(cached.CheckedAt) < updateCheckInterval {
			return cached.LatestVersion, nil
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateCheckClient.Timeout)
	defer cancel()

	latest, err := latestFn(ctx)
	if err != nil {
		if cached, ok := readUpdateCache(); ok {
			return cached.LatestVersion, nil
		}
		return "", err
	}
	writeUpdateCache(updateCache{CheckedAt: now, LatestVersion: latest})
	return latest, nil
}

// updateNotice returns the lines of a user-facing notice when a release newer
// than currentVersion is available, or nil. It performs the daily cached check
// and never returns an error: a broken check must not disrupt the actual command.
func updateNotice(currentVersion string, now time.Time, latestFn func(context.Context) (string, error)) []string {
	if currentVersion == "" || currentVersion == "dev" {
		return nil
	}
	latest, err := cachedLatestVersion(now, latestFn)
	if err != nil || latest == "" {
		return nil
	}
	if !isNewer(latest, currentVersion) {
		return nil
	}
	return []string{
		fmt.Sprintf("a new version of sproot is available: %s (you have %s)", displayVersion(latest), displayVersion(currentVersion)),
		"run 'sproot self-update' to upgrade",
	}
}

// NotifyUpdateAvailable prints a once-per-day notice to stderr when a newer
// sproot release exists. It is best-effort: any failure is silently ignored.
// Set SPROOT_NO_UPDATE_CHECK (to any value) to disable it, e.g. in CI.
func NotifyUpdateAvailable(currentVersion string) {
	if os.Getenv(noUpdateCheckEnv) != "" {
		return
	}
	lines := updateNotice(currentVersion, time.Now(), func(ctx context.Context) (string, error) {
		return fetchLatestVersion(ctx, updateCheckClient)
	})
	if len(lines) == 0 {
		return
	}
	l := log.Stderr()
	for _, line := range lines {
		l.Warn(line)
	}
}

// SelfUpdateOptions controls RunSelfUpdate. CurrentVersion is the running
// binary's version (main.version). CheckOnly reports availability without
// downloading or replacing anything. The remaining fields are test seams.
type SelfUpdateOptions struct {
	CurrentVersion string
	CheckOnly      bool

	latestFn      func(context.Context) (string, error)                     // nil: query GitHub
	fetchFn       func(ctx context.Context, version string) ([]byte, error) // nil: download+extract for this OS/arch
	execPath      string                                                    // empty: resolve via os.Executable
	brewUpgradeFn func(ctx context.Context) error                           // nil: run real `brew upgrade`
}

// RunSelfUpdate upgrades the running sproot binary to the latest GitHub release.
// It always queries upstream (ignoring the daily cache) so an explicit upgrade is
// never served stale data, and it clears the update-check cache after a
// successful upgrade so the next command reflects the new state.
func RunSelfUpdate(ctx context.Context, opts SelfUpdateOptions) error {
	l := log.Stderr()

	dev := opts.CurrentVersion == "dev" || opts.CurrentVersion == ""
	if dev && !opts.CheckOnly {
		return fmt.Errorf(
			"self-update is not supported for dev builds; install a release with the install script: " +
				"https://github.com/justanotherspy/sproot#installation",
		)
	}

	// Resolve the binary we would replace up front so a Homebrew-managed install
	// can be detected before any network access.
	dest := opts.execPath
	if dest == "" {
		exe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("locate running binary: %w", err)
		}
		if resolved, err := filepath.EvalSymlinks(exe); err == nil {
			exe = resolved
		}
		dest = exe
	}

	// A Homebrew cask install must be upgraded through brew: overwriting the
	// Caskroom binary in place would leave brew's version tracking out of sync.
	// --check still uses the normal GitHub check below.
	if !opts.CheckOnly && isHomebrewManaged(dest) {
		brewUpgrade := opts.brewUpgradeFn
		if brewUpgrade == nil {
			brewUpgrade = runBrewUpgrade
		}
		l.Info("sproot was installed with Homebrew; upgrading via 'brew upgrade --cask sproot'")
		if err := brewUpgrade(ctx); err != nil {
			return err
		}
		clearUpdateCache()
		return nil
	}

	latestFn := opts.latestFn
	if latestFn == nil {
		latestFn = func(ctx context.Context) (string, error) {
			return fetchLatestVersion(ctx, binaryDownloadClient)
		}
	}

	l.Info("checking for the latest sproot release")
	latest, err := latestFn(ctx)
	if err != nil {
		return fmt.Errorf("check latest release: %w", err)
	}

	upgradeAvailable := !dev && isNewer(latest, opts.CurrentVersion)

	// Check-only, dev build, or already current: refresh the cache so the daily
	// notifier reflects this check, report status, and stop.
	if opts.CheckOnly || !upgradeAvailable {
		writeUpdateCache(updateCache{CheckedAt: time.Now(), LatestVersion: latest})
		switch {
		case dev:
			l.Infof("latest sproot release is %s (you are running a dev build)", displayVersion(latest))
		case upgradeAvailable:
			l.Warnf("a new version of sproot is available: %s (you have %s)", displayVersion(latest), displayVersion(opts.CurrentVersion))
			l.Warn("run 'sproot self-update' to upgrade")
		default:
			l.Successf("sproot is already up to date (%s)", displayVersion(opts.CurrentVersion))
		}
		return nil
	}

	l.Infof("upgrading sproot %s -> %s", displayVersion(opts.CurrentVersion), displayVersion(latest))

	fetchFn := opts.fetchFn
	if fetchFn == nil {
		fetchFn = func(ctx context.Context, version string) ([]byte, error) {
			return downloadReleaseBinary(ctx, version, runtime.GOOS, runtime.GOARCH)
		}
	}
	binaryData, err := fetchFn(ctx, latest)
	if err != nil {
		return fmt.Errorf("download release: %w", err)
	}

	if err := replaceExecutable(dest, binaryData); err != nil {
		return err
	}

	// Clear the cache so the next command re-checks against the new version.
	clearUpdateCache()
	l.Successf("upgraded sproot to %s (%s)", displayVersion(latest), dest)
	return nil
}

// isHomebrewManaged reports whether the resolved binary path lives inside a
// Homebrew prefix (a cask under .../Caskroom/... or a formula under
// .../Cellar/...), in which case self-update defers to `brew upgrade`.
func isHomebrewManaged(execPath string) bool {
	resolved := execPath
	if r, err := filepath.EvalSymlinks(execPath); err == nil {
		resolved = r
	}
	sep := string(os.PathSeparator)
	return strings.Contains(resolved, sep+"Caskroom"+sep) ||
		strings.Contains(resolved, sep+"Cellar"+sep)
}

// runBrewUpgrade upgrades the sproot cask through Homebrew, streaming brew's
// output to the user's terminal.
func runBrewUpgrade(ctx context.Context) error {
	brew, err := exec.LookPath("brew")
	if err != nil {
		return fmt.Errorf(
			"sproot was installed with Homebrew but 'brew' is not on PATH; " +
				"upgrade manually with 'brew upgrade --cask sproot'",
		)
	}
	cmd := exec.CommandContext(ctx, brew, "upgrade", "--cask", "sproot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("brew upgrade --cask sproot: %w", err)
	}
	return nil
}

// downloadReleaseBinary downloads the release archive for version and the host's
// OS/arch, verifies it against the published checksums file, and returns the
// extracted sproot binary bytes. version may carry a leading "v".
func downloadReleaseBinary(ctx context.Context, version, goos, goarch string) ([]byte, error) {
	bare := strings.TrimPrefix(version, "v")
	tag := "v" + bare

	archive := releaseArchiveName(bare, goos, goarch)
	checksums := fmt.Sprintf("sproot_%s_checksums.txt", bare)
	base := fmt.Sprintf("%s/%s", releaseDownloadBase, tag)

	archiveBytes, err := downloadBytes(ctx, base+"/"+archive)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", archive, err)
	}
	checksumBytes, err := downloadBytes(ctx, base+"/"+checksums)
	if err != nil {
		return nil, fmt.Errorf("download %s: %w", checksums, err)
	}
	if err := verifyChecksum(archiveBytes, checksumBytes, archive); err != nil {
		return nil, err
	}
	return extractBinary(archiveBytes, goos)
}

func releaseArchiveName(bareVersion, goos, goarch string) string {
	if goos == "windows" {
		return fmt.Sprintf("sproot_%s_%s_%s.zip", bareVersion, goos, goarch)
	}
	return fmt.Sprintf("sproot_%s_%s_%s.tar.gz", bareVersion, goos, goarch)
}

func downloadBytes(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "sproot")

	resp, err := binaryDownloadClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// verifyChecksum confirms archiveBytes matches its sha256 entry in a
// goreleaser-style checksums file (lines of "<hex>  <filename>").
func verifyChecksum(archiveBytes, checksumsFile []byte, archiveName string) error {
	sum := sha256.Sum256(archiveBytes)
	got := hex.EncodeToString(sum[:])

	var want string
	for _, line := range strings.Split(string(checksumsFile), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[len(fields)-1] == archiveName {
			want = fields[0]
			break
		}
	}
	if want == "" {
		return fmt.Errorf("no checksum entry for %s", archiveName)
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("checksum mismatch for %s: got %s, want %s", archiveName, got, want)
	}
	return nil
}

func extractBinary(archiveBytes []byte, goos string) ([]byte, error) {
	if goos == "windows" {
		return extractSprootFromZip(archiveBytes)
	}
	return extractSprootFromTarGz(bytes.NewReader(archiveBytes))
}

// extractSprootFromZip returns the bytes of the sproot binary entry (sproot or
// sproot.exe, at any depth) within a zip archive held in memory.
func extractSprootFromZip(archiveBytes []byte) ([]byte, error) {
	zr, err := zip.NewReader(bytes.NewReader(archiveBytes), int64(len(archiveBytes)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}
	for _, f := range zr.File {
		name := f.Name
		if name == "sproot.exe" || strings.HasSuffix(name, "/sproot.exe") ||
			name == "sproot" || strings.HasSuffix(name, "/sproot") {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("open zip entry: %w", err)
			}
			data, err := io.ReadAll(rc)
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("read zip entry: %w", err)
			}
			return data, nil
		}
	}
	return nil, fmt.Errorf("sproot binary not found in archive")
}

// replaceExecutable swaps the file at dest with a new binary built from data.
// On Unix it renames a sibling temp file over dest (the running process keeps the
// old, now-unlinked inode). On Windows, where the running image cannot be
// overwritten, it moves the current binary aside first.
func replaceExecutable(dest string, data []byte) error {
	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".sproot-update-*")
	if err != nil {
		return fmt.Errorf(
			"cannot write to %s (self-update needs write access to the install directory; "+
				"re-run with sufficient privileges or reinstall via the install script): %w",
			dir, err,
		)
	}
	tmpName := tmp.Name()
	cleanup := func() { _ = os.Remove(tmpName) }

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		cleanup()
		return fmt.Errorf("write new binary: %w", err)
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return fmt.Errorf("write new binary: %w", err)
	}
	if err := os.Chmod(tmpName, 0o755); err != nil {
		cleanup()
		return fmt.Errorf("chmod new binary: %w", err)
	}

	if runtime.GOOS == "windows" {
		old := dest + ".old"
		_ = os.Remove(old)
		if err := os.Rename(dest, old); err != nil {
			cleanup()
			return fmt.Errorf("move current binary aside: %w", err)
		}
		if err := os.Rename(tmpName, dest); err != nil {
			_ = os.Rename(old, dest) // best-effort rollback
			cleanup()
			return fmt.Errorf("install new binary: %w", err)
		}
		_ = os.Remove(old) // may be locked while running; ignore
		return nil
	}

	if err := os.Rename(tmpName, dest); err != nil {
		cleanup()
		return fmt.Errorf(
			"install new binary at %s (self-update needs write access to the install directory; "+
				"re-run with sufficient privileges or reinstall via the install script): %w",
			dest, err,
		)
	}
	return nil
}
