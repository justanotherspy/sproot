package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newSelfUpdateCmd() *cobra.Command {
	var checkOnly bool
	cmd := &cobra.Command{
		Use:     "self-update",
		Aliases: []string{"selfupdate", "self-upgrade"},
		Short:   "Upgrade the sproot binary itself to the latest release",
		Long: "self-update downloads the latest sproot release for your OS and " +
			"architecture, verifies its checksum, and replaces the running binary. " +
			"It always checks upstream (ignoring the daily update cache) and clears " +
			"that cache after upgrading. If sproot was installed with Homebrew, it " +
			"runs 'brew upgrade --cask sproot' instead so brew stays in sync. Pass " +
			"--check to report whether an update is available without installing it. " +
			"To upgrade the sprite CLI inside a sprite, use 'sproot upgrade <name>' instead.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunSelfUpdate(cmd.Context(), host.SelfUpdateOptions{
				CurrentVersion: version,
				CheckOnly:      checkOnly,
			})
		},
	}
	cmd.Flags().BoolVar(&checkOnly, "check", false, "report whether a newer release is available without installing it")
	return cmd
}
