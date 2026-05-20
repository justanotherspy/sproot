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

func TestTemplateAsset(t *testing.T) {
	version := "v2.4.0"
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
			"{goos}_{dpkg_arch}",
			func(s string) bool { return strings.Contains(s, runtime.GOOS) },
			"goos substituted",
		},
		{
			"notemplates",
			func(s string) bool { return s == "notemplates" },
			"no substitution when no vars",
		},
	}
	for _, tc := range cases {
		got := templateAsset(tc.pattern, version)
		if !tc.check(got) {
			t.Errorf("templateAsset(%q, %q) = %q: %s", tc.pattern, version, got, tc.desc)
		}
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
