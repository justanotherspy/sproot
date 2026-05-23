package host

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/pkg/log"
)

// configCache caches the resolved sproot.yaml content for each (repo, ref, path)
// keyed by the git commit SHA the ref pointed at. Repeated host-side operations
// (sproot new, push, outdated) can then skip the full git clone when the remote
// has not advanced, checking only a cheap `git ls-remote` instead.
//
// Only the public sproot.yaml content is stored. Secret env values are never
// written to disk: the env block is re-resolved from the host environment on
// every call (loadConfigBytes returns content; readEnvBlock resolves secrets).
type configCache struct {
	Entries map[string]configCacheEntry `json:"entries"`
}

type configCacheEntry struct {
	CommitSHA string `json:"commit_sha"` // git commit SHA the ref pointed at when cached
	RawConfig string `json:"raw_config"` // sproot.yaml bytes at that commit
}

func configCacheKey(repo, ref, configPath string) string {
	if configPath == "" {
		configPath = "sproot.yaml"
	}
	return strings.Join([]string{repo, ref, configPath}, "\x00")
}

func configCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sproot", "config-cache.json"), nil
}

func readConfigCache() configCache {
	empty := configCache{Entries: map[string]configCacheEntry{}}
	path, err := configCachePath()
	if err != nil {
		return empty
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty
	}
	var c configCache
	if err := json.Unmarshal(data, &c); err != nil || c.Entries == nil {
		return empty
	}
	return c
}

// writeConfigCache persists c atomically (temp file plus rename). Errors are
// ignored: the cache is an optimization, not a source of truth.
func writeConfigCache(c configCache) {
	path, err := configCachePath()
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
	tmp, err := os.CreateTemp(dir, ".config-cache-*")
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

// lsRemoteFn probes a remote with git ls-remote. It is a package variable so
// tests can stub the network call in effectiveCloneURL.
var lsRemoteFn = gitLsRemote

// effectiveCloneURL returns the URL to use for host-side resolution of repo at
// ref. For a GitHub SSH URL it keeps SSH when SSH works (preserving private-repo
// access via the host's own key), and falls back to the HTTPS form only when SSH
// is unreachable (e.g. no SSH key on this host), so a public config repo still
// resolves. Non-GitHub-SSH URLs are returned unchanged with no probing.
//
// This mirrors the in-sprite bootstrap rewrite (see cloneOrPull): the in-sprite
// clone always rewrites to HTTPS, so passing along the original SSH URL stays
// correct regardless of which form resolves here.
func effectiveCloneURL(repo, ref string, l *log.Logger) string {
	alt, rewritten := config.NormalizeGitHubCloneURL(repo)
	if !rewritten {
		return repo
	}
	if _, err := lsRemoteFn(repo, ref); err == nil {
		return repo
	}
	if _, err := lsRemoteFn(alt, ref); err == nil {
		l.Warnf("config repo %q is not reachable over SSH from this host; resolving %q over HTTPS instead; set sproot_config_repo to the HTTPS URL to silence this", repo, alt)
		return alt
	}
	return repo
}

// gitLsRemote returns the commit SHA that ref resolves to in repo using a single
// lightweight `git ls-remote` (no clone, no checkout).
func gitLsRemote(repo, ref string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.GitOpTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "git", "ls-remote", repo, ref).Output() // nosemgrep
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("git ls-remote timed out after %s", config.GitOpTimeout)
		}
		return "", fmt.Errorf("git ls-remote: %w", err)
	}
	return parseLsRemote(string(out), ref)
}

// parseLsRemote picks the commit SHA for ref from `git ls-remote` output. When
// ref matches both a branch and a tag, the branch is preferred (mirroring
// `git clone --branch`). For annotated tags the dereferenced commit (the "^{}"
// line) is preferred over the tag object SHA so it matches `git rev-parse HEAD`
// after a clone.
func parseLsRemote(out, ref string) (string, error) {
	var head, tagDeref, tagObj, first string
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		sha, name := fields[0], fields[1]
		if first == "" {
			first = sha
		}
		switch name {
		case "refs/heads/" + ref:
			head = sha
		case "refs/tags/" + ref + "^{}":
			tagDeref = sha
		case "refs/tags/" + ref:
			tagObj = sha
		}
	}
	switch {
	case head != "":
		return head, nil
	case tagDeref != "":
		return tagDeref, nil
	case tagObj != "":
		return tagObj, nil
	case first != "":
		return first, nil
	default:
		return "", fmt.Errorf("ref %q not found on remote", ref)
	}
}

// loadConfigBytes returns the raw sproot.yaml bytes for (repo, ref, configPath).
// It first tries a cheap `git ls-remote`; on a cache hit (the ref still points at
// the cached commit) it returns the cached content without cloning. Otherwise it
// performs the full shallow clone, refreshes the cache, and returns the freshly
// read bytes. An ls-remote failure is non-fatal: the function falls back to a
// clone so behavior is never worse than before the cache existed.
func loadConfigBytes(repo, ref, configPath string, l *log.Logger) ([]byte, error) {
	repo = effectiveCloneURL(repo, ref, l)
	key := configCacheKey(repo, ref, configPath)

	if remoteSHA, err := gitLsRemote(repo, ref); err == nil {
		cache := readConfigCache()
		if entry, ok := cache.Entries[key]; ok && entry.CommitSHA == remoteSHA {
			l.Debugf("config cache hit for %s@%s (%s); skipping clone", repo, ref, remoteSHA)
			return []byte(entry.RawConfig), nil
		}
		l.Debugf("config cache miss for %s@%s; cloning", repo, ref)
		raw, commitSHA, err := cloneAndRead(repo, ref, configPath, l)
		if err != nil {
			return nil, err
		}
		if commitSHA == "" {
			commitSHA = remoteSHA
		}
		cache.Entries[key] = configCacheEntry{CommitSHA: commitSHA, RawConfig: string(raw)}
		writeConfigCache(cache)
		return raw, nil
	}

	l.Debugf("git ls-remote unavailable for %s@%s; cloning", repo, ref)
	raw, commitSHA, err := cloneAndRead(repo, ref, configPath, l)
	if err != nil {
		return nil, err
	}
	if commitSHA != "" {
		cache := readConfigCache()
		cache.Entries[key] = configCacheEntry{CommitSHA: commitSHA, RawConfig: string(raw)}
		writeConfigCache(cache)
	}
	return raw, nil
}

// cloneAndRead shallow-clones repo at ref into a temp dir, reads the config file
// at configPath (or "sproot.yaml" when empty), and returns its bytes plus the
// checked-out commit SHA (empty if rev-parse fails).
func cloneAndRead(repo, ref, configPath string, l *log.Logger) ([]byte, string, error) {
	tmpdir, err := os.MkdirTemp("", "sproot-config-*")
	if err != nil {
		return nil, "", fmt.Errorf("create temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpdir) }()

	l.Infof("cloning config repo to resolve sproot.yaml")
	ctx, cancel := context.WithTimeout(context.Background(), config.GitOpTimeout)
	defer cancel()
	cloneArgs := []string{"clone", "--depth", "1", "--branch", ref, repo, tmpdir}
	if out, err := exec.CommandContext(ctx, "git", cloneArgs...).CombinedOutput(); err != nil { // nosemgrep
		if ctx.Err() == context.DeadlineExceeded {
			return nil, "", fmt.Errorf("git clone timed out after %s: %w", config.GitOpTimeout, ctx.Err())
		}
		return nil, "", fmt.Errorf("git clone: %w\n%s", err, out)
	}

	yamlFile := configPath
	if yamlFile == "" {
		yamlFile = "sproot.yaml"
	}
	raw, err := os.ReadFile(filepath.Join(tmpdir, yamlFile))
	if err != nil {
		return nil, "", fmt.Errorf("read %s: %w", yamlFile, err)
	}

	commitSHA := ""
	if out, err := exec.CommandContext(ctx, "git", "-C", tmpdir, "rev-parse", "HEAD").Output(); err == nil { // nosemgrep
		commitSHA = strings.TrimSpace(string(out))
	}
	return raw, commitSHA, nil
}
