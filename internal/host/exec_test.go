package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunExec_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunExec(context.Background(), ExecOptions{Name: "my-sprite", Cmd: "ls"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunExec_RunsCommandWithArgs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
config_repo: git@github.com:user/repo.git
config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunExec(context.Background(), ExecOptions{
		Name:   "my-sprite",
		Cmd:    "echo",
		Args:   []string{"hello", "world"},
		client: client,
	})
	if err != nil {
		t.Fatalf("RunExec: %v", err)
	}
	if handle.lastCmdName != "echo" {
		t.Errorf("expected cmd echo, got %q", handle.lastCmdName)
	}
	if len(handle.lastCmdArgs) != 2 || handle.lastCmdArgs[0] != "hello" {
		t.Errorf("unexpected args: %v", handle.lastCmdArgs)
	}
}
