package host

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"testing"
)

func makeTarGz(t *testing.T, entries map[string][]byte) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, content := range entries {
		hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return &buf
}

func TestExtractSprootFromTarGz_TopLevel(t *testing.T) {
	want := []byte("fake-binary-content")
	buf := makeTarGz(t, map[string][]byte{
		"sproot":    want,
		"README.md": []byte("readme"),
	})
	got, err := extractSprootFromTarGz(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("content mismatch: got %q, want %q", got, want)
	}
}

func TestExtractSprootFromTarGz_WithDirectoryPrefix(t *testing.T) {
	want := []byte("fake-binary-content")
	buf := makeTarGz(t, map[string][]byte{
		"sproot_1.2.3_linux_amd64/sproot":    want,
		"sproot_1.2.3_linux_amd64/README.md": []byte("readme"),
	})
	got, err := extractSprootFromTarGz(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("content mismatch: got %q, want %q", got, want)
	}
}

func TestExtractSprootFromTarGz_NotFound(t *testing.T) {
	buf := makeTarGz(t, map[string][]byte{
		"README.md": []byte("readme"),
		"LICENSE":   []byte("license"),
	})
	_, err := extractSprootFromTarGz(buf)
	if err == nil {
		t.Fatal("expected error when sproot entry is absent")
	}
}

func TestLinuxAmd64ReleaseURL(t *testing.T) {
	// A "v"-prefixed and bare version must yield the same asset name (goreleaser
	// {{ .Version }} has no "v") while the release tag always carries the "v".
	const wantName = "sproot_1.2.3_linux_amd64.tar.gz"
	const wantURL = "https://github.com/justanotherspy/sproot/releases/download/v1.2.3/sproot_1.2.3_linux_amd64.tar.gz"
	for _, version := range []string{"1.2.3", "v1.2.3"} {
		name, url := linuxAmd64ReleaseURL(version)
		if name != wantName {
			t.Errorf("linuxAmd64ReleaseURL(%q) name = %q, want %q", version, name, wantName)
		}
		if url != wantURL {
			t.Errorf("linuxAmd64ReleaseURL(%q) url = %q, want %q", version, url, wantURL)
		}
	}
}

func TestFetchLinuxAmd64Binary_DevVersion(t *testing.T) {
	_, err := fetchLinuxAmd64Binary("dev")
	if err == nil {
		t.Fatal("expected error for dev version")
	}
	errMsg := err.Error()
	if len(errMsg) == 0 || errMsg == "dev" {
		t.Errorf("error should be descriptive, got: %v", err)
	}
}
