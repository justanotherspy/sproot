package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newSpriteStatusCmd() *cobra.Command {
	var fromHost bool
	cmd := &cobra.Command{
		Use:   "status <name>",
		Short: "Print phase status for a sprite",
		Long: "status prints the sproot phase status for the named sprite.\n\n" +
			"Without --host it runs 'sproot setup --status' inside the sprite (requires exec).\n" +
			"With --host it reads the state file from the sprite filesystem directly, without exec.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunStatus(cmd.Context(), host.StatusOptions{
				Name: args[0],
				Host: fromHost,
			})
		},
	}
	cmd.Flags().BoolVar(&fromHost, "host", false, "read state from sprite filesystem without exec")
	return cmd
}
