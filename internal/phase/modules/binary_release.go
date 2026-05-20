package modules

// binary_release downloads and installs a GitHub release asset.
//
// Asset name supports template variables:
//   - {version}   — the release tag (e.g. v2.4.0)
//   - {arch}      — Go arch (amd64, arm64)
//   - {goos}      — Go OS   (linux, darwin)
//   - {dpkg_arch} — Debian arch (amd64, arm64)
//
// Install methods: dpkg, tar+install, raw.
//
//	- type: binary_release
//	  name: cosign
//	  repo: sigstore/cosign
//	  asset: "cosign_{version}_{arch}.deb"
//	  install: dpkg

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
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

	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/phase"
	"github.com/justanotherspy/sproot/pkg/log"
)

var (
	// tagClient is used for short GitHub API calls (tag lookup, checksums file).
	tagClient = &http.Client{Timeout: 30 * time.Second}
	// downloadClient is used for asset downloads which can be large.
	downloadClient = &http.Client{Timeout: 5 * time.Minute}
)

func init() {
	phase.Register("binary_release", func(cfg config.PhaseConfig) (phase.Phase, error) {
		if cfg.BinaryRelease == nil {
			return nil, fmt.Errorf("binary_release: config is nil")
		}
		return &binaryReleasePhase{cfg: cfg.BinaryRelease}, nil
	})
}

type binaryReleasePhase struct {
	cfg *config.BinaryReleaseConfig
}

func (p *binaryReleasePhase) Type() string { return "binary_release" }
func (p *binaryReleasePhase) Name() string {
	return fmt.Sprintf("binary_release(%s)", p.cfg.Name)
}

func (p *binaryReleasePhase) ShouldRun(_ *phase.Context) (bool, error) {
	if p.cfg.Install == "dpkg" {
		return !checkCmd("dpkg", "-s", p.cfg.Name), nil
	}
	_, err := exec.LookPath(p.cfg.Name)
	return err != nil, nil
}

func (p *binaryReleasePhase) Run(ctx *phase.Context) error {
	version, err := githubLatestTag(p.cfg.Repo)
	if err != nil {
		return fmt.Errorf("binary_release(%s): get latest tag: %w", p.cfg.Name, err)
	}
	ctx.Log.Infof("latest %s: %s", p.cfg.Name, version)

	assetName := templateAsset(p.cfg.Asset, version)
	url := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", p.cfg.Repo, version, assetName)
	ctx.Log.Infof("downloading %s", url)

	tmp, err := downloadAsset(url)
	if err != nil {
		return fmt.Errorf("binary_release(%s): download: %w", p.cfg.Name, err)
	}
	defer func() { _ = os.Remove(tmp) }()

	if p.cfg.Checksum != "" {
		if err := verifyChecksum(tmp, p.cfg.Checksum); err != nil {
			return fmt.Errorf("binary_release(%s): checksum: %w", p.cfg.Name, err)
		}
	}
	if p.cfg.ChecksumAsset != "" {
		checksumAssetName := templateAsset(p.cfg.ChecksumAsset, version)
		checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", p.cfg.Repo, version, checksumAssetName)
		if err := verifyChecksumAsset(tmp, assetName, checksumURL); err != nil {
			return fmt.Errorf("binary_release(%s): checksum_asset: %w", p.cfg.Name, err)
		}
	}

	switch p.cfg.Install {
	case "dpkg":
		return runCmd(ctx.Log, "dpkg", "-i", tmp)
	case "tar+install":
		return installFromTar(ctx.Log, tmp, p.cfg.Name)
	case "raw":
		return installRaw(tmp, p.cfg.Name)
	default:
		return fmt.Errorf("binary_release(%s): unknown install method %q", p.cfg.Name, p.cfg.Install)
	}
}

func (p *binaryReleasePhase) Verify(_ *phase.Context) error {
	if p.cfg.Install == "dpkg" {
		if !checkCmd("dpkg", "-s", p.cfg.Name) {
			return fmt.Errorf("binary_release(%s): package not installed", p.cfg.Name)
		}
		return nil
	}
	if _, err := exec.LookPath(p.cfg.Name); err != nil {
		return fmt.Errorf("binary_release(%s): not on PATH after install", p.cfg.Name)
	}
	return nil
}

// githubLatestTag returns the latest release tag for owner/repo.
func githubLatestTag(repo string) (string, error) {
	apiURL := "https://api.github.com/repos/" + repo + "/releases/latest"
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	if tok := os.Getenv("GH_TOKEN"); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := tagClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", err
	}
	if rel.TagName == "" {
		return "", fmt.Errorf("no tag_name in response")
	}
	return rel.TagName, nil
}

// templateAsset replaces template variables in the asset name pattern.
func templateAsset(pattern, version string) string {
	arch := runtime.GOARCH
	dpkgArch := arch // amd64/arm64 map directly for Debian
	r := strings.NewReplacer(
		"{version}", version,
		"{arch}", arch,
		"{goos}", runtime.GOOS,
		"{dpkg_arch}", dpkgArch,
	)
	return r.Replace(pattern)
}

// downloadAsset downloads url to a temp file and returns its path.
func downloadAsset(url string) (string, error) {
	resp, err := downloadClient.Get(url) //nolint:noctx
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}
	f, err := os.CreateTemp("", "sproot-asset-*")
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(f, resp.Body); err != nil {
		_ = f.Close()
		_ = os.Remove(f.Name())
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(f.Name())
		return "", err
	}
	return f.Name(), nil
}

// installFromTar extracts a .tar.gz, finds the named binary, and installs it to /usr/local/bin.
func installFromTar(l *log.Logger, tarPath, name string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip open: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	dest := "/usr/local/bin/" + name
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar read: %w", err)
		}
		if filepath.Base(hdr.Name) != name {
			continue
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
		if err != nil {
			return fmt.Errorf("create %s: %w", dest, err)
		}
		if _, err := io.Copy(out, tr); err != nil {
			_ = out.Close()
			return fmt.Errorf("write %s: %w", dest, err)
		}
		if err := out.Close(); err != nil {
			return fmt.Errorf("close %s: %w", dest, err)
		}
		l.Infof("installed %s", dest)
		return nil
	}
	return fmt.Errorf("binary %q not found in tarball", name)
}

// verifyChecksum computes the sha256 of the file at path and compares it to
// the expected hex string. Returns an error if they do not match.
func verifyChecksum(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	got := hex.EncodeToString(h.Sum(nil))
	if got != strings.ToLower(want) {
		return fmt.Errorf("sha256 mismatch: got %s, want %s", got, want)
	}
	return nil
}

// verifyChecksumAsset downloads a goreleaser-style checksums file from checksumURL,
// finds the line for assetName, and verifies the downloaded file at path.
// Expected line format: "<sha256hex>  <filename>"
func verifyChecksumAsset(path, assetName, checksumURL string) error {
	resp, err := tagClient.Get(checksumURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download checksums file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums file: HTTP %d for %s", resp.StatusCode, checksumURL)
	}
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.SplitN(line, "  ", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == assetName {
			return verifyChecksum(path, parts[0])
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read checksums file: %w", err)
	}
	return fmt.Errorf("asset %q not found in checksums file", assetName)
}

// installRaw copies the downloaded file to /usr/local/bin/<name> with execute permission.
func installRaw(src, name string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	dest := "/usr/local/bin/" + name
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("create %s: %w", dest, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
