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
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunExec(context.Background(), ExecOptions{Name: "my-sprite", Cmd: "ls"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunExec_ForwardsEnv(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunExec(context.Background(), ExecOptions{
		Name:   "my-sprite",
		Cmd:    "env",
		Env:    "FOO=bar,BAZ=qux",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunExec: %v", err)
	}
	if len(handle.lastCmdEnv) != 2 {
		t.Errorf("expected 2 env vars, got %v", handle.lastCmdEnv)
	}
	found := map[string]bool{"FOO=bar": false, "BAZ=qux": false}
	for _, e := range handle.lastCmdEnv {
		found[e] = true
	}
	for k, v := range found {
		if !v {
			t.Errorf("env var %q not forwarded; got %v", k, handle.lastCmdEnv)
		}
	}
}

func TestRunExec_RunsCommandWithArgs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
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
