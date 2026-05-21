package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newOutdatedCmd() *cobra.Command {
	var opts host.OutdatedOptions
	cmd := &cobra.Command{
		Use:   "outdated",
		Short: "Check which sproot-managed sprites have a stale config",
		Long: "outdated fetches the current config SHA and compares it against the " +
			"sproot-sha label stored on each sprite. Sprites whose stored SHA differs " +
			"from the current config are reported as stale. Run 'sproot push' to " +
			"update them.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunOutdated(cmd.Context(), opts)
		},
	}
	return cmd
}
