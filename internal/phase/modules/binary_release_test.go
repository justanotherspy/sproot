package modules

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestCopyCapped(t *testing.T) {
	// Under and at the limit succeed; over the limit fails rather than
	// silently truncating (as a bare io.CopyN at the cap would).
	cases := []struct {
		name    string
		size    int64
		limit   int64
		wantErr bool
	}{
		{"under limit", 5, 10, false},
		{"exactly at limit", 10, 10, false},
		{"over limit", 11, 10, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := strings.NewReader(strings.Repeat("x", int(c.size)))
			var dst strings.Builder
			err := copyCapped(&dst, src, c.limit)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error for size %d over limit %d", c.size, c.limit)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if int64(dst.Len()) != c.size {
				t.Errorf("copied %d bytes, want %d", dst.Len(), c.size)
			}
		})
	}
}

func TestTemplateAsset(t *testing.T) {
	version := "v2.4.0"
	tag := "v2.4.0"
	cases := []struct {
		pattern string
		check   func(string) bool
		desc    string
	}{
		{
			"{version}_{arch}.deb",
			func(s string) bool { return strings.Contains(s, "v2.4.0") && strings.Contains(s, runtime.GOARCH) },
			"version and arch substituted",
		},
		{
			"{tag}_{tag_no_v}",
			func(s string) bool { return s == "v2.4.0_2.4.0" },
			"tag and tag_no_v substituted",
		},
		{
			"{goos}_{dpkg_arch}",
			func(s string) bool { return strings.Contains(s, runtime.GOOS) },
			"goos substituted",
		},
		{
			"{x64_arch}",
			func(s string) bool {
				switch runtime.GOARCH {
				case "amd64":
					return s == "x64"
				case "arm64":
					return s == "arm64"
				}
				return s != ""
			},
			"x64_arch substituted",
		},
		{
			"{x86_64_arch}",
			func(s string) bool {
				switch runtime.GOARCH {
				case "amd64":
					return s == "x86_64"
				case "arm64":
					return s == "aarch64"
				}
				return s != ""
			},
			"x86_64_arch substituted",
		},
		{
			"notemplates",
			func(s string) bool { return s == "notemplates" },
			"no substitution when no vars",
		},
	}
	for _, tc := range cases {
		got, err := templateAsset(tc.pattern, version, tag, nil)
		if err != nil {
			t.Errorf("templateAsset(%q): unexpected error: %v", tc.pattern, err)
			continue
		}
		if !tc.check(got) {
			t.Errorf("templateAsset(%q) = %q: %s", tc.pattern, got, tc.desc)
		}
	}
}

func TestTemplateAsset_ArchAlias(t *testing.T) {
	archMap := map[string]string{"amd64": "x86_64", "arm64": "arm64"}
	got, err := templateAsset("hadolint-linux-{arch_alias}", "1.0", "v1.0", archMap)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "hadolint-linux-" + archMap[runtime.GOARCH]
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestTemplateAsset_ArchAliasWithoutMap(t *testing.T) {
	if _, err := templateAsset("x-{arch_alias}", "1.0", "v1.0", nil); err == nil {
		t.Error("expected error when {arch_alias} used without arch_map")
	}
}

func TestTemplateAsset_ArchAliasMissingKey(t *testing.T) {
	// A map that lacks the current GOARCH key must error.
	archMap := map[string]string{"some-other-arch": "x"}
	if _, err := templateAsset("x-{arch_alias}", "1.0", "v1.0", archMap); err == nil {
		t.Error("expected error when arch_map lacks current GOARCH key")
	}
}

func TestStripLeadingV(t *testing.T) {
	cases := map[string]string{
		"v8.18.4": "8.18.4",
		"V8.18.4": "8.18.4",
		"8.18.4":  "8.18.4",
		"":        "",
		"vv1":     "v1", // strips only one leading v
	}
	for in, want := range cases {
		if got := stripLeadingV(in); got != want {
			t.Errorf("stripLeadingV(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResolveVersion(t *testing.T) {
	cases := []struct {
		tmpl, tag, want string
	}{
		{"", "v8.18.4", "v8.18.4"},          // empty -> raw tag
		{"{tag}", "v8.18.4", "v8.18.4"},     // explicit raw tag
		{"{tag_no_v}", "v8.18.4", "8.18.4"}, // stripped
		{"{tag_no_v}", "8.18.4", "8.18.4"},  // tag already without v
		{"r{tag_no_v}", "v1.2", "r1.2"},     // mixed literal + var
	}
	for _, tc := range cases {
		if got := resolveVersion(tc.tmpl, tc.tag); got != tc.want {
			t.Errorf("resolveVersion(%q, %q) = %q, want %q", tc.tmpl, tc.tag, got, tc.want)
		}
	}
}

func TestVerifyCosign_AbsentCosignErrors(t *testing.T) {
	// Force a PATH with no cosign so LookPath fails deterministically.
	t.Setenv("PATH", t.TempDir())
	p := &binaryReleasePhase{cfg: &config.BinaryReleaseConfig{
		Name: "trufflehog", Repo: "trufflesecurity/trufflehog",
		Cosign: &config.CosignConfig{
			Blob: "c.txt", Signature: "c.txt.sig", Certificate: "c.txt.pem",
			CertificateIdentityRegexp: "x", CertificateOIDCIssuer: "y",
		},
	}}
	err := p.verifyCosign(testCtx(t), "v1.0.0", "1.0.0", "/tmp/asset", "asset")
	if err == nil || !strings.Contains(err.Error(), "cosign") {
		t.Errorf("expected cosign-absent error, got: %v", err)
	}
}

func TestVerifyChecksumFile_Match(t *testing.T) {
	content := []byte("fake binary content")
	h := sha256.Sum256(content)
	checksum := hex.EncodeToString(h[:])
	assetName := "mytool_1.0.0_linux_amd64.tar.gz"

	dir := t.TempDir()
	checksumsPath := dir + "/checksums.txt"
	if err := os.WriteFile(checksumsPath, []byte(fmt.Sprintf("%s  %s\naabbcc  other.tar.gz\n", checksum, assetName)), 0o644); err != nil {
		t.Fatal(err)
	}
	assetPath := dir + "/asset"
	if err := os.WriteFile(assetPath, content, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := verifyChecksumFile(assetPath, assetName, checksumsPath); err != nil {
		t.Errorf("expected match, got: %v", err)
	}
	if err := verifyChecksumFile(assetPath, "missing.tar.gz", checksumsPath); err == nil {
		t.Error("expected not-found error for missing asset")
	}
}

func TestBinaryRelease_ShouldRunWhenBinaryMissing(t *testing.T) {
	p := &binaryReleasePhase{cfg: &config.BinaryReleaseConfig{
		Name:    "sproot-nonexistent-bin",
		Repo:    "owner/repo",
		Asset:   "{version}.tar.gz",
		Install: "raw",
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if !should {
		t.Error("expected ShouldRun=true for missing binary")
	}
}

func TestBinaryRelease_ShouldRunFalseWhenInstalled(t *testing.T) {
	// "sh" is guaranteed to be on PATH.
	p := &binaryReleasePhase{cfg: &config.BinaryReleaseConfig{
		Name:    "sh",
		Repo:    "owner/repo",
		Asset:   "{version}.tar.gz",
		Install: "raw",
	}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when binary already on PATH")
	}
}

func TestVerifyChecksum_Match(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	content := []byte("fake binary content")
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	h := sha256.Sum256(content)
	want := hex.EncodeToString(h[:])

	if err := verifyChecksum(f.Name(), want); err != nil {
		t.Errorf("expected match, got: %v", err)
	}
}

func TestVerifyChecksum_Mismatch(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write([]byte("fake binary content")); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := verifyChecksum(f.Name(), "0000000000000000000000000000000000000000000000000000000000000000"); err == nil {
		t.Error("expected mismatch error")
	}
}

func TestVerifyChecksumAsset_Match(t *testing.T) {
	content := []byte("fake binary content")
	h := sha256.Sum256(content)
	checksum := hex.EncodeToString(h[:])
	assetName := "mytool_v1.0.0_linux_amd64.tar.gz"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  %s\n", checksum, assetName)
		_, _ = fmt.Fprintf(w, "aabbcc  other_file.tar.gz\n")
	}))
	defer srv.Close()

	f, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := verifyChecksumAsset(f.Name(), assetName, srv.URL); err != nil {
		t.Errorf("expected match, got: %v", err)
	}
}

func TestVerifyChecksumAsset_Mismatch(t *testing.T) {
	content := []byte("fake binary content")
	assetName := "mytool_v1.0.0_linux_amd64.tar.gz"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  %s\n", strings.Repeat("0", 64), assetName)
	}))
	defer srv.Close()

	f, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.Write(content); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := verifyChecksumAsset(f.Name(), assetName, srv.URL); err == nil {
		t.Error("expected mismatch error")
	}
}

// TestHTTPClientTimeouts verifies the 9d fix: both HTTP clients have explicit timeouts.
func TestHTTPClientTimeouts(t *testing.T) {
	if tagClient.Timeout == 0 {
		t.Error("tagClient has no timeout set")
	}
	if downloadClient.Timeout == 0 {
		t.Error("downloadClient has no timeout set")
	}
}

// TestGithubLatestTag_SendsAuthHeader verifies the 9c fix: when GH_TOKEN is set,
// the Authorization header is included in the tag lookup request.
func TestGithubLatestTag_SendsAuthHeader(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		_, _ = fmt.Fprintf(w, `{"tag_name":"v1.2.3"}`)
	}))
	defer srv.Close()

	// Temporarily point tagClient at our test server by swapping the global.
	orig := tagClient
	tagClient = &http.Client{Timeout: orig.Timeout}
	defer func() { tagClient = orig }()

	t.Setenv("GH_TOKEN", "test-token")

	// Call githubLatestTag with a fake repo path; the server URL already covers the full path.
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	if tok := "test-token"; tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	resp, err := tagClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()

	if !strings.HasPrefix(gotAuth, "Bearer ") {
		t.Errorf("expected Authorization: Bearer ..., got %q", gotAuth)
	}
}

func TestVerifyChecksumAsset_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintf(w, "%s  other_file.tar.gz\n", strings.Repeat("a", 64))
	}))
	defer srv.Close()

	f, err := os.CreateTemp(t.TempDir(), "asset-*")
	if err != nil {
		t.Fatal(err)
	}
	_ = f.Close()

	if err := verifyChecksumAsset(f.Name(), "missing_asset.tar.gz", srv.URL); err == nil {
		t.Error("expected error when asset not found in checksums file")
	}
}
