package modules

// binary_release downloads and installs a GitHub release asset.
//
// Asset name supports template variables:
//   - {version}     — resolved version token (defaults to the raw tag; see version field)
//   - {tag}         — the raw release tag (e.g. v2.4.0)
//   - {tag_no_v}    — the tag with one leading v/V stripped (e.g. 2.4.0)
//   - {arch}        — Go arch (amd64, arm64)
//   - {goos}        — Go OS   (linux, darwin)
//   - {dpkg_arch}   — Debian arch (amd64, arm64)
//   - {x64_arch}    — x64-style arch (x64 on amd64, arm64 on arm64)
//   - {x86_64_arch} — x86_64-style arch (x86_64 on amd64, aarch64 on arm64)
//   - {arch_alias}  — arch_map[GOARCH] (for naming schemes the fixed vars miss)
//
// The download URL path always uses the raw tag, independent of {version}.
// Install methods: dpkg, tar+install, raw.
// An optional cosign block verifies a Sigstore keyless signature before install.
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
	tag, err := githubLatestTag(p.cfg.Repo)
	if err != nil {
		return fmt.Errorf("binary_release(%s): get latest tag: %w", p.cfg.Name, err)
	}
	ctx.Log.Infof("latest %s: %s", p.cfg.Name, tag)

	// version is the token substituted into asset names; the download URL path
	// always uses the raw tag.
	version := resolveVersion(p.cfg.Version, tag)

	assetName, err := templateAsset(p.cfg.Asset, version, tag, p.cfg.ArchMap)
	if err != nil {
		return fmt.Errorf("binary_release(%s): asset: %w", p.cfg.Name, err)
	}
	url := p.assetURL(tag, assetName)
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
	if p.cfg.Cosign != nil {
		if err := p.verifyCosign(ctx, tag, version, tmp, assetName); err != nil {
			return fmt.Errorf("binary_release(%s): cosign: %w", p.cfg.Name, err)
		}
	}
	if p.cfg.ChecksumAsset != "" {
		checksumAssetName, err := templateAsset(p.cfg.ChecksumAsset, version, tag, p.cfg.ArchMap)
		if err != nil {
			return fmt.Errorf("binary_release(%s): checksum_asset: %w", p.cfg.Name, err)
		}
		checksumURL := p.assetURL(tag, checksumAssetName)
		if err := verifyChecksumAsset(tmp, assetName, checksumURL); err != nil {
			return fmt.Errorf("binary_release(%s): checksum_asset: %w", p.cfg.Name, err)
		}
	}

	switch p.cfg.Install {
	case "dpkg":
		return runPrivileged(ctx.Log, "dpkg", "-i", tmp)
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

// assetURL builds a release download URL. The path always uses the raw tag.
func (p *binaryReleasePhase) assetURL(tag, assetName string) string {
	return fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", p.cfg.Repo, tag, assetName)
}

// verifyCosign downloads the signed blob, its signature, and its certificate,
// then runs `cosign verify-blob` (keyless). On success, if a checksums file is
// being verified, it also verifies the main asset against that trusted blob.
func (p *binaryReleasePhase) verifyCosign(ctx *phase.Context, tag, version, assetTmp, assetName string) error {
	if _, err := exec.LookPath("cosign"); err != nil {
		return fmt.Errorf("cosign verification configured but cosign is not on PATH (install it via an earlier binary_release phase)")
	}
	c := p.cfg.Cosign
	blobTmp, err := p.downloadCosignAsset(c.Blob, tag, version)
	if err != nil {
		return fmt.Errorf("download blob: %w", err)
	}
	defer func() { _ = os.Remove(blobTmp) }()
	sigTmp, err := p.downloadCosignAsset(c.Signature, tag, version)
	if err != nil {
		return fmt.Errorf("download signature: %w", err)
	}
	defer func() { _ = os.Remove(sigTmp) }()
	certTmp, err := p.downloadCosignAsset(c.Certificate, tag, version)
	if err != nil {
		return fmt.Errorf("download certificate: %w", err)
	}
	defer func() { _ = os.Remove(certTmp) }()

	if err := runCmd(ctx.Log, "cosign", "verify-blob",
		"--certificate", certTmp,
		"--signature", sigTmp,
		"--certificate-identity-regexp", c.CertificateIdentityRegexp,
		"--certificate-oidc-issuer", c.CertificateOIDCIssuer,
		blobTmp); err != nil {
		return fmt.Errorf("verify-blob: %w", err)
	}

	// The verified blob is typically a checksums file; verify the main asset
	// against it so the download is bound to the trusted signature.
	if err := verifyChecksumFile(assetTmp, assetName, blobTmp); err != nil {
		return fmt.Errorf("verify asset against signed checksums: %w", err)
	}
	return nil
}

// downloadCosignAsset templates a cosign asset name and downloads it to a temp file.
func (p *binaryReleasePhase) downloadCosignAsset(tmplName, tag, version string) (string, error) {
	name, err := templateAsset(tmplName, version, tag, p.cfg.ArchMap)
	if err != nil {
		return "", err
	}
	return downloadAsset(p.assetURL(tag, name))
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

// stripLeadingV removes a single leading v or V from a version tag.
func stripLeadingV(s string) string {
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}
	return s
}

// resolveVersion resolves the version token used in asset names. An empty tmpl
// returns the raw tag (preserving legacy behavior); otherwise {tag} and
// {tag_no_v} are substituted so the config can choose whether to keep the v.
func resolveVersion(tmpl, tag string) string {
	if tmpl == "" {
		return tag
	}
	return strings.NewReplacer(
		"{tag}", tag,
		"{tag_no_v}", stripLeadingV(tag),
	).Replace(tmpl)
}

// templateAsset replaces template variables in an asset name pattern. version is
// the already-resolved version token; tag is the raw release tag. archMap maps
// GOARCH to a custom arch token exposed as {arch_alias}. It returns an error when
// {arch_alias} is used without a matching arch_map entry.
func templateAsset(pattern, version, tag string, archMap map[string]string) (string, error) {
	arch := runtime.GOARCH
	dpkgArch := arch // amd64/arm64 map directly for Debian
	var x64Arch, x8664Arch string
	switch arch {
	case "amd64":
		x64Arch = "x64"
		x8664Arch = "x86_64"
	case "arm64":
		x64Arch = "arm64"
		x8664Arch = "aarch64"
	default:
		x64Arch = arch
		x8664Arch = arch
	}
	archAlias := ""
	if strings.Contains(pattern, "{arch_alias}") {
		if len(archMap) == 0 {
			return "", fmt.Errorf("asset %q uses {arch_alias} but arch_map is not set", pattern)
		}
		a, ok := archMap[arch]
		if !ok {
			return "", fmt.Errorf("arch_map has no entry for GOARCH %q (asset %q)", arch, pattern)
		}
		archAlias = a
	}
	r := strings.NewReplacer(
		"{version}", version,
		"{tag}", tag,
		"{tag_no_v}", stripLeadingV(tag),
		"{arch}", arch,
		"{goos}", runtime.GOOS,
		"{dpkg_arch}", dpkgArch,
		"{x64_arch}", x64Arch,
		"{x86_64_arch}", x8664Arch,
		"{arch_alias}", archAlias,
	)
	return r.Replace(pattern), nil
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
		const maxBinarySize = 500 << 20 // 500 MB; guards against decompression bombs
		if err := copyCapped(out, tr, maxBinarySize); err != nil {
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

// copyCapped copies src to dst, failing if src holds more than limit bytes (a
// decompression-bomb guard). A source of exactly limit bytes is allowed. Unlike
// a bare io.CopyN, this returns an error rather than silently truncating when
// the source exceeds the cap.
func copyCapped(dst io.Writer, src io.Reader, limit int64) error {
	n, err := io.CopyN(dst, src, limit+1)
	if err != nil && err != io.EOF {
		return err
	}
	if n > limit {
		return fmt.Errorf("exceeds %d byte limit", limit)
	}
	return nil
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
func verifyChecksumAsset(path, assetName, checksumURL string) error {
	resp, err := tagClient.Get(checksumURL) //nolint:noctx
	if err != nil {
		return fmt.Errorf("download checksums file: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksums file: HTTP %d for %s", resp.StatusCode, checksumURL)
	}
	return verifyChecksumFromReader(path, assetName, resp.Body)
}

// verifyChecksumFile verifies the file at path against a local goreleaser-style
// checksums file. Used after the checksums file itself has been cosign-verified.
func verifyChecksumFile(path, assetName, checksumsPath string) error {
	f, err := os.Open(checksumsPath)
	if err != nil {
		return fmt.Errorf("open checksums file: %w", err)
	}
	defer func() { _ = f.Close() }()
	return verifyChecksumFromReader(path, assetName, f)
}

// verifyChecksumFromReader scans a checksums stream (lines "<sha256hex>  <filename>")
// for assetName and verifies the file at path against the listed hash.
func verifyChecksumFromReader(path, assetName string, r io.Reader) error {
	scanner := bufio.NewScanner(r)
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
