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
