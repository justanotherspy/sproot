package host

import (
	"context"
	"fmt"
	"os"

	"github.com/justanotherspy/sproot/internal/config"
)

// ExecOptions controls the behavior of RunExec.
type ExecOptions struct {
	Name    string
	Cmd     string
	Args    []string
	client  SpritesClient // nil: constructed from token at runtime
}

// RunExec runs a command in the named sprite and streams stdout/stderr to the host.
func RunExec(ctx context.Context, opts ExecOptions) error {
	cfgPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return err
	}
	cfg, err := loadOrInitHostConfig(cfgPath)
	if err != nil {
		return err
	}
	if err := config.ValidateHostConfig(cfg); err != nil {
		return err
	}

	token := os.Getenv(cfg.TokenEnv)
	if token == "" {
		return fmt.Errorf("env var %s (token_env) is not set", cfg.TokenEnv)
	}

	client := opts.client
	if client == nil {
		client = NewClient(token)
	}

	handle := client.GetHandle(opts.Name)
	return handle.RunCommand(opts.Cmd, opts.Args, nil, os.Stdout, os.Stderr)
}
