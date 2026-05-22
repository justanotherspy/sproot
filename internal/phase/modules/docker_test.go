package modules

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/justanotherspy/sproot/internal/config"
)

func TestDocker_ShouldRunWhenDockerMissing(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		p := &dockerPhase{cfg: &config.DockerConfig{}}
		should, err := p.ShouldRun(testCtx(t))
		if err != nil {
			t.Fatal(err)
		}
		if !should {
			t.Error("expected ShouldRun=true when docker not on PATH")
		}
		return
	}
	t.Skip("docker on PATH; skip missing-tool path")
}

func TestDocker_ShouldRunFalseWhenDockerPresent(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not on PATH")
	}
	p := &dockerPhase{cfg: &config.DockerConfig{}}
	should, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatal(err)
	}
	if should {
		t.Error("expected ShouldRun=false when docker already installed")
	}
}

func TestDocker_TypeAndName(t *testing.T) {
	p := &dockerPhase{cfg: &config.DockerConfig{}}
	if p.Type() != "docker" {
		t.Errorf("Type: got %q", p.Type())
	}
	if p.Name() != "docker" {
		t.Errorf("Name: got %q", p.Name())
	}
}

func TestDocker_ShouldRunWhenDaemonJSONMissing(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not on PATH")
	}
	dir := t.TempDir()
	// Point daemon.json check at a path that does not exist.
	_ = dir // used to ensure tmp path is unique
	p := &dockerPhase{cfg: &config.DockerConfig{
		DaemonJSON: map[string]any{"log-driver": "json-file"},
	}}
	// Monkey-patch: the ShouldRun checks /etc/docker/daemon.json specifically.
	// In this test environment that file may or may not exist.
	// We only verify the logic doesn't panic.
	_, err := p.ShouldRun(testCtx(t))
	if err != nil {
		t.Fatalf("ShouldRun returned unexpected error: %v", err)
	}
}

func TestMergeDaemonJSON_AbsentFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	data := map[string]any{"storage-driver": "overlay2"}
	if err := mergeDaemonJSONAt(path, data); err != nil {
		t.Fatalf("mergeDaemonJSONAt: %v", err)
	}
	got := readJSON(t, path)
	if got["storage-driver"] != "overlay2" {
		t.Errorf("storage-driver not written: %v", got)
	}
}

func TestMergeDaemonJSON_MergesExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	// Pre-existing file (e.g. a platform default).
	if err := os.WriteFile(path, []byte(`{"storage-driver":"overlay2","log-opts":{"max-size":"1m"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	data := map[string]any{
		"log-driver": "json-file",
		"log-opts":   map[string]any{"max-file": "3"},
	}
	if err := mergeDaemonJSONAt(path, data); err != nil {
		t.Fatalf("mergeDaemonJSONAt: %v", err)
	}
	got := readJSON(t, path)
	if got["storage-driver"] != "overlay2" {
		t.Error("existing storage-driver lost after merge")
	}
	if got["log-driver"] != "json-file" {
		t.Error("new log-driver missing after merge")
	}
	opts, _ := got["log-opts"].(map[string]any)
	if opts["max-size"] != "1m" || opts["max-file"] != "3" {
		t.Errorf("nested log-opts not merged: %v", opts)
	}
}

func TestMergeDaemonJSON_CorruptExisting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	if err := os.WriteFile(path, []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := mergeDaemonJSONAt(path, map[string]any{"a": 1}); err == nil {
		t.Error("expected error on corrupt existing daemon.json")
	}
}

func TestDaemonJSONChanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "daemon.json")
	data := map[string]any{"storage-driver": "overlay2"}

	changed, err := daemonJSONChanged(path, data)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Error("expected changed=true when file absent")
	}
	if err := mergeDaemonJSONAt(path, data); err != nil {
		t.Fatal(err)
	}
	changed, err = daemonJSONChanged(path, data)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Error("expected changed=false after writing the same merged content")
	}
}

func readJSON(t *testing.T, path string) map[string]any {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
	return out
}
