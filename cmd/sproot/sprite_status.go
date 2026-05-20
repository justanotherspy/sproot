package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newSpriteStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Print phase status for a running sprite",
		Long:  "status runs 'sproot setup --status' inside the named sprite and streams the output.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunStatus(cmd.Context(), host.StatusOptions{Name: args[0]})
		},
	}
	return cmd
}
