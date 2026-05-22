package modules

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/justanotherspy/sproot/pkg/log"
)

// runPrivileged runs name+args with root privilege. When the current process is
// not root (the usual case inside a sprite, where setup runs as the unprivileged
// sprite user) and sudo is available, it prepends `sudo -n`. On a host already
// running as root, or where sudo is absent, it runs the command directly. Use it
// for operations that categorically require root (dpkg, apt-get install).
func runPrivileged(l *log.Logger, name string, args ...string) error {
	if os.Geteuid() != 0 {
		if sudo, err := exec.LookPath("sudo"); err == nil {
			return runCmd(l, sudo, append([]string{"-n", name}, args...)...)
		}
	}
	return runCmd(l, name, args...)
}

// runCmd logs and executes name with args, streaming stdout+stderr to log line by line.
func runCmd(l *log.Logger, name string, args ...string) error {
	l.Debugf("exec: %s %s", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...) // nosemgrep
	pr, pw, err := pipeOf(cmd)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return fmt.Errorf("%s: %w", name, err)
	}
	_ = pw.Close()
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		l.Info(scanner.Text())
	}
	_ = pr.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// runCmdIn is like runCmd but sets the working directory before executing.
func runCmdIn(dir string, l *log.Logger, name string, args ...string) error {
	l.Debugf("exec (in %s): %s %s", dir, name, strings.Join(args, " "))
	cmd := exec.Command(name, args...) // nosemgrep
	cmd.Dir = dir
	pr, pw, err := pipeOf(cmd)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return fmt.Errorf("%s: %w", name, err)
	}
	_ = pw.Close()
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		l.Info(scanner.Text())
	}
	_ = pr.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// runCmdEnv is like runCmd but appends extra "KEY=value" entries to the process
// environment for this command only (e.g. GOBIN for go install).
func runCmdEnv(env []string, l *log.Logger, name string, args ...string) error {
	l.Debugf("exec (env %s): %s %s", strings.Join(env, " "), name, strings.Join(args, " "))
	cmd := exec.Command(name, args...) // nosemgrep
	cmd.Env = append(os.Environ(), env...)
	pr, pw, err := pipeOf(cmd)
	if err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	if err := cmd.Start(); err != nil {
		_ = pw.Close()
		_ = pr.Close()
		return fmt.Errorf("%s: %w", name, err)
	}
	_ = pw.Close()
	scanner := bufio.NewScanner(pr)
	for scanner.Scan() {
		l.Info(scanner.Text())
	}
	_ = pr.Close()
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

// userLocalBin returns ~/.local/bin (expanded), creating it if needed. It is the
// install target for tools whose default location is not on the sprite's PATH
// (go install -> GOPATH/bin, corepack-managed pnpm/yarn -> node bin dir);
// ~/.local/bin is on the default PATH, so installed tools become invocable.
func userLocalBin() (string, error) {
	dir, err := expandTilde("~/.local/bin")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// userLocalRoot returns ~/.local (expanded), creating it if needed. It is the
// --root for `cargo install`, which writes binaries to <root>/bin and tracks
// them in <root>/.crates2.json. cargo's default root (CARGO_HOME/bin) is not on
// the sprite's PATH, whereas ~/.local/bin is, so installed crates stay invocable.
func userLocalRoot() (string, error) {
	dir, err := expandTilde("~/.local")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create %s: %w", dir, err)
	}
	return dir, nil
}

// pipeOf wires a single pipe to cmd stdout+stderr and returns the read end.
func pipeOf(cmd *exec.Cmd) (io.ReadCloser, io.WriteCloser, error) {
	pr, pw, err := os.Pipe()
	if err != nil {
		return nil, nil, err
	}
	cmd.Stdout = pw
	cmd.Stderr = pw
	return pr, pw, nil
}

// checkCmd returns true if name with args exits 0.
func checkCmd(name string, args ...string) bool {
	cmd := exec.Command(name, args...) // nosemgrep
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run() == nil
}

// outputOf runs name with args and returns trimmed stdout. Stderr is discarded.
func outputOf(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output() // nosemgrep
	if err != nil {
		return "", fmt.Errorf("%s: %w", name, err)
	}
	return strings.TrimSpace(string(out)), nil
}
