package main

import (
	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a sproot.yaml file",
		Long:  "validate loads a sproot.yaml and reports any missing or invalid fields. Defaults to the config_path from ~/.sproot/config.yaml, or sproot.yaml in the current directory.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				path = resolveDefaultSprootPath()
			}
			return host.RunValidateSprootConfig(path)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "path to sproot.yaml (default: config_path from ~/.sproot/config.yaml, or sproot.yaml)")
	return cmd
}

// resolveDefaultSprootPath returns the config_path from the host config if readable,
// falling back to "sproot.yaml" in the current directory.
func resolveDefaultSprootPath() string {
	hostPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return "sproot.yaml"
	}
	hcfg, err := config.LoadHostConfig(hostPath)
	if err != nil || hcfg.ConfigPath == "" {
		return "sproot.yaml"
	}
	return hcfg.ConfigPath
}
