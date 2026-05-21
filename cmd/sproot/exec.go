package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec <name> <cmd> [args...]",
		Short: "Run a command in a sprite and stream output",
		Long:  "exec runs a single command in the named sprite and streams stdout/stderr to the host. Non-interactive.",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunExec(cmd.Context(), host.ExecOptions{
				Name: args[0],
				Cmd:  args[1],
				Args: args[2:],
			})
		},
	}
	return cmd
}
