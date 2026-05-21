package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var all bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprites created by sproot",
		Long:  "list shows sprites tagged with the sproot label. Use --all to show all sprites regardless of label.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunList(cmd.Context(), host.ListOptions{All: all})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show all sprites, not just sproot-managed ones")
	return cmd
}
