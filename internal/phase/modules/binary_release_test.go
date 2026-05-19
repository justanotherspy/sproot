package modules

import (
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
