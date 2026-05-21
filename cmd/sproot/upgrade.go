package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newUpgradeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upgrade <name>",
		Short: "Upgrade the sprite CLI inside a sprite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunUpgrade(cmd.Context(), host.UpgradeOptions{Name: args[0]})
		},
	}
	return cmd
}
