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
			"using the config repo from ~/.sproot/config. After successful setup it opens " +
			"an interactive console. Use --no-console to skip that step.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			opts.Version = version
			return host.RunNew(cmd.Context(), opts)
		},
	}
	cmd.Flags().IntVar(&opts.RamMB, "ram-mb", 0, "RAM in MB for the sprite (0 uses API default)")
	cmd.Flags().IntVar(&opts.CPUs, "cpus", 0, "CPU count for the sprite (0 uses API default)")
	cmd.Flags().StringVar(&opts.Region, "region", "", "region for the sprite (empty uses API default)")
	cmd.Flags().IntVar(&opts.StorageGB, "storage-gb", 0, "storage in GB for the sprite (0 uses API default)")
	cmd.Flags().StringVar(&opts.ConfigPath, "config-path", "", "path to config file within the repo (overrides host config)")
	cmd.Flags().StringVar(&opts.Only, "only", "", "run only phases matching this type")
	cmd.Flags().BoolVar(&opts.Force, "force", false, "re-run phases even if already complete")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "describe changes without executing them")
	cmd.Flags().BoolVar(&opts.NoConsole, "no-console", false, "skip opening a console after setup completes")
	return cmd
}
