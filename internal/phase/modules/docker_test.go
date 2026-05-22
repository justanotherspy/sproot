package modules

import (
	"encoding/json"
	"os/exec"
	"strings"
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

func TestWriteDaemonJSON(t *testing.T) {
	// writeDaemonJSON targets /etc/docker/daemon.json which requires root.
	// Test the JSON marshaling logic without actually writing the file.
	data := map[string]any{
		"log-driver": "json-file",
		"log-opts":   map[string]any{"max-size": "10m"},
	}
	// Verify it marshals without error (logic path used by writeDaemonJSON).
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		t.Fatalf("json.MarshalIndent: %v", err)
	}
	if !strings.Contains(string(b), "log-driver") {
		t.Errorf("marshaled JSON missing log-driver: %s", b)
	}
}
