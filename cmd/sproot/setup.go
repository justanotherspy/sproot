package main

import (
	"github.com/justanotherspy/sproot/internal/sprite"
	"github.com/spf13/cobra"
)

func newSetupCmd() *cobra.Command {
	var opts sprite.SetupOptions
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Clone config repo and run phases inside a sprite",
		Long: "setup clones the config repo, loads sproot.yaml, and runs all phases " +
			"to provision the sprite environment. Pass --status to print a summary " +
			"of the last run instead of executing phases.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return sprite.RunSetup(opts)
		},
	}
	cmd.Flags().StringVar(&opts.ConfigRepo, "config-repo", "", "git URL of the config repo")
	cmd.Flags().StringVar(&opts.Ref, "ref", "main", "branch, tag, or SHA to check out")
	cmd.Flags().StringVar(&opts.ConfigPath, "config-path", "", "path to config file within the repo (default: sproot.yaml)")
	cmd.Flags().StringVar(&opts.Only, "only", "", "run only phases matching this type")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "re-run phases even if already complete")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "describe changes without executing them")
	cmd.Flags().BoolVar(&opts.Status, "status", false, "print phase status table and exit")
	return cmd
}
