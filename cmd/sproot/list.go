package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var all bool
	var prefix string
	var watch bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List sprites created by sproot",
		Long:  "list shows sprites tagged with the sproot label. Use --all to show all sprites regardless of label, --prefix to filter by name prefix, or --watch to refresh continuously.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunList(cmd.Context(), host.ListOptions{All: all, Prefix: prefix, Watch: watch})
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "show all sprites, not just sproot-managed ones")
	cmd.Flags().StringVar(&prefix, "prefix", "", "filter sprites by name prefix")
	cmd.Flags().BoolVar(&watch, "watch", false, "watch for live updates (polls every 2s)")
	return cmd
}
