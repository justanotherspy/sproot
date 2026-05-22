package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newPushCmd() *cobra.Command {
	var opts host.PushOptions
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Re-run setup on all sproot-managed sprites",
		Long: "push checkpoints and then re-runs sproot setup --force on all sprites " +
			"created by sproot new (identified by the sproot label). Use --name to target " +
			"a single sprite. Sprites are updated in parallel.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return host.RunPush(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.SpriteName, "name", "", "push to a single named sprite (default: all sproot-managed sprites)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "pass --target to sproot setup")
	cmd.Flags().StringVar(&opts.Only, "only", "", "pass --only to sproot setup (run only the named phase type)")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "describe changes without executing them")
	cmd.Flags().BoolVar(&opts.NoCheckpoint, "no-checkpoint", false, "skip pre-push checkpoint")
	return cmd
}
