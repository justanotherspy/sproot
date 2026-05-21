package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newCheckpointCmd() *cobra.Command {
	var comment string
	cmd := &cobra.Command{
		Use:   "checkpoint <name>",
		Short: "Create a checkpoint for a sprite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunCheckpoint(cmd.Context(), host.CheckpointOptions{
				Name:    args[0],
				Comment: comment,
			})
		},
	}
	cmd.Flags().StringVar(&comment, "comment", "", "optional comment for the checkpoint")
	return cmd
}

func newCheckpointsCmd() *cobra.Command {
	var includeAuto bool
	cmd := &cobra.Command{
		Use:   "checkpoints <name>",
		Short: "List checkpoints for a sprite",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunCheckpoints(cmd.Context(), host.CheckpointsOptions{
				Name:        args[0],
				IncludeAuto: includeAuto,
			})
		},
	}
	cmd.Flags().BoolVar(&includeAuto, "include-auto", false, "include auto-created checkpoints")
	return cmd
}

func newRestoreCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restore <name> <checkpoint-id>",
		Short: "Restore a sprite from a checkpoint",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunRestore(cmd.Context(), host.RestoreOptions{
				Name:         args[0],
				CheckpointID: args[1],
			})
		},
	}
	return cmd
}
