package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var opts host.NewOptions
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new sprite and run sproot setup inside it",
		Long: "new creates a sprite, injects the sproot binary, and runs sproot setup " +
			"using the config repo from ~/.sproot/config.yaml. After successful setup it opens " +
			"an interactive console. Use --skip-console to skip that step.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			opts.Version = version
			return host.RunNew(cmd.Context(), opts)
		},
	}
	cmd.Flags().StringVar(&opts.ConfigPath, "config-path", "", "path to config file within the repo (overrides host config)")
	cmd.Flags().StringVar(&opts.Target, "target", "", "named target to run from sproot.yaml (requires targets: block)")
	cmd.Flags().StringVar(&opts.LocalConfig, "local-config", "", "host directory to use as config source instead of git clone")
	cmd.Flags().StringVar(&opts.Only, "only", "", "run only phases matching this type")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "re-run phases even if already complete")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "describe changes without executing them")
	cmd.Flags().BoolVar(&opts.SkipConsole, "skip-console", false, "skip opening a console after setup completes")
	cmd.Flags().BoolVar(&opts.SkipVerify, "skip-verify", false, "skip the built-in verify phase")
	return cmd
}
