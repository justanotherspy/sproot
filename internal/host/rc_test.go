package host

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func rcHostConfig(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "fly-tok")
}

func argsContain(args []string, want string) bool {
	for _, a := range args {
		if a == want {
			return true
		}
	}
	return false
}

func TestRunRC_TokenEnvUnset(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeHostConfig(t, filepath.Join(home, ".sproot"), `
sproot_config_repo: git@github.com:user/repo.git
sproot_config_ref: main
token_env: MY_TOKEN
`)
	t.Setenv("MY_TOKEN", "")

	err := RunRC(context.Background(), RCOptions{Name: "my-sprite"})
	if err == nil || !strings.Contains(err.Error(), "MY_TOKEN") {
		t.Errorf("expected token env error, got %v", err)
	}
}

func TestRunRC_NotAuthenticated(t *testing.T) {
	rcHostConfig(t)

	handle := newMockHandle() // no creds file -> ReadFile returns not-exist
	client := &mockClient{handle: handle}

	err := RunRC(context.Background(), RCOptions{Name: "my-sprite", client: client})
	if err == nil || !strings.Contains(err.Error(), "not authenticated") {
		t.Fatalf("expected not-authenticated error, got %v", err)
	}
	if handle.lastCmdName != "" {
		t.Errorf("expected no sprite command when unauthenticated, got %q", handle.lastCmdName)
	}
	if _, ok := handle.writtenFiles[rcScriptPath]; ok {
		t.Error("expected no launcher script written when unauthenticated")
	}
}

func TestRunRC_StartRegistersService(t *testing.T) {
	rcHostConfig(t)

	handle := newMockHandle()
	handle.readFiles[rcCredsPath] = []byte("{}")
	client := &mockClient{handle: handle}

	err := RunRC(context.Background(), RCOptions{
		Name:   "my-sprite",
		Dir:    "~/repos/foo",
		Spawn:  "worktree",
		client: client,
	})
	if err != nil {
		t.Fatalf("RunRC: %v", err)
	}

	script, ok := handle.writtenFiles[rcScriptPath]
	if !ok {
		t.Fatalf("expected launcher script written to %s", rcScriptPath)
	}
	s := string(script)
	for _, want := range []string{
		"claude remote-control",
		`--name "my-sprite"`,
		"--spawn worktree",
		`cd "$HOME/repos/foo"`,
	} {
		if !strings.Contains(s, want) {
			t.Errorf("launcher script missing %q; got:\n%s", want, s)
		}
	}

	if handle.lastCmdName != "sprite-env" {
		t.Errorf("expected sprite-env command, got %q", handle.lastCmdName)
	}
	if !argsContain(handle.lastCmdArgs, "PUT") || !argsContain(handle.lastCmdArgs, "/v1/services/"+rcServiceName) {
		t.Errorf("expected PUT to the service path, got args %v", handle.lastCmdArgs)
	}
}

func TestRunRC_StartDefaultsSessionNameAndDir(t *testing.T) {
	rcHostConfig(t)

	handle := newMockHandle()
	handle.readFiles[rcCredsPath] = []byte("{}")
	client := &mockClient{handle: handle}

	if err := RunRC(context.Background(), RCOptions{Name: "my-sprite", client: client}); err != nil {
		t.Fatalf("RunRC: %v", err)
	}

	s := string(handle.writtenFiles[rcScriptPath])
	if !strings.Contains(s, `--name "my-sprite"`) {
		t.Errorf("expected session title to default to sprite name; got:\n%s", s)
	}
	if !strings.Contains(s, `cd "$HOME"`) {
		t.Errorf("expected working dir to default to $HOME; got:\n%s", s)
	}
	if !strings.Contains(s, "--spawn same-dir") {
		t.Errorf("expected default spawn mode same-dir; got:\n%s", s)
	}
}

func TestRunRC_InvalidSpawn(t *testing.T) {
	rcHostConfig(t)

	handle := newMockHandle()
	handle.readFiles[rcCredsPath] = []byte("{}")
	client := &mockClient{handle: handle}

	err := RunRC(context.Background(), RCOptions{Name: "my-sprite", Spawn: "bogus", client: client})
	if err == nil || !strings.Contains(err.Error(), "invalid --spawn") {
		t.Fatalf("expected invalid spawn error, got %v", err)
	}
	if handle.lastCmdName != "" {
		t.Errorf("expected no sprite command for invalid spawn, got %q", handle.lastCmdName)
	}
}

func TestRunRC_CloseDeletesService(t *testing.T) {
	rcHostConfig(t)

	handle := newMockHandle()
	client := &mockClient{handle: handle}

	err := RunRC(context.Background(), RCOptions{Name: "my-sprite", Close: true, client: client})
	if err != nil {
		t.Fatalf("RunRC --close: %v", err)
	}
	if handle.lastCmdName != "sprite-env" {
		t.Errorf("expected sprite-env command, got %q", handle.lastCmdName)
	}
	if !argsContain(handle.lastCmdArgs, "DELETE") || !argsContain(handle.lastCmdArgs, "/v1/services/"+rcServiceName) {
		t.Errorf("expected DELETE to the service path, got args %v", handle.lastCmdArgs)
	}
	if _, ok := handle.writtenFiles[rcScriptPath]; ok {
		t.Error("expected no launcher script written on close")
	}
}
