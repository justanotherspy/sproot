package host

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var binaryDownloadClient = &http.Client{Timeout: 5 * time.Minute}

// fetchLinuxAmd64Binary downloads the Linux/amd64 sproot binary for the given
// version from GitHub releases and returns the raw executable bytes.
// version must be a released version string (e.g. "1.2.3" or "v1.2.3");
// passing "dev" returns an error.
func fetchLinuxAmd64Binary(version string) ([]byte, error) {
	if version == "dev" {
		return nil, fmt.Errorf(
			"cannot inject Linux/amd64 binary: running a dev build has no matching release; " +
				"build sproot for linux/amd64 or run sproot from a linux/amd64 host",
		)
	}

	// Normalize version: goreleaser sets main.version without leading "v",
	// but the GitHub release tag includes it.
	tag := version
	if !strings.HasPrefix(version, "v") {
		tag = "v" + version
	}

	archiveName := fmt.Sprintf("sproot_%s_linux_amd64.tar.gz", version)
	url := fmt.Sprintf(
		"https://github.com/justanotherspy/sproot/releases/download/%s/%s",
		tag, archiveName,
	)

	resp, err := binaryDownloadClient.Get(url) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("download linux/amd64 binary: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download linux/amd64 binary: HTTP %d from %s", resp.StatusCode, url)
	}

	data, err := extractSprootFromTarGz(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("extract from %s: %w", archiveName, err)
	}
	return data, nil
}

// extractSprootFromTarGz reads a .tar.gz stream and returns the bytes of the
// entry named "sproot" (at any directory depth).
func extractSprootFromTarGz(r io.Reader) ([]byte, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("open gzip: %w", err)
	}
	defer func() { _ = gz.Close() }()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar: %w", err)
		}
		// Match "sproot" at top level or inside a subdirectory.
		name := hdr.Name
		if name == "sproot" || strings.HasSuffix(name, "/sproot") {
			return io.ReadAll(tr)
		}
	}
	return nil, fmt.Errorf("sproot binary not found in archive")
}
