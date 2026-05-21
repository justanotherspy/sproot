package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newConsoleCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "console <name>",
		Short: "Open an interactive shell on a sprite",
		Long:  "console opens a TTY shell on the named sprite. The sprite must already exist.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunConsole(cmd.Context(), host.ConsoleOptions{Name: args[0]})
		},
	}
}
