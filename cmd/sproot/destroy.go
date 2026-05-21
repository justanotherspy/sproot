package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newDestroyCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "destroy <name>",
		Short: "Remove GitHub SSH keys and destroy a sprite",
		Long: "destroy reads the GitHub key IDs written by ssh_setup, removes them from " +
			"the GitHub account, then destroys the sprite. Key deletion failures are " +
			"logged as warnings and do not block the destroy. Prompts for confirmation " +
			"unless --force is set.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunDestroy(cmd.Context(), host.DestroyOptions{Name: args[0], Force: force})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "skip confirmation prompt")
	return cmd
}
