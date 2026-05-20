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

// runCmd logs and executes name with args, streaming stdout+stderr to log line by line.
func runCmd(l *log.Logger, name string, args ...string) error {
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
