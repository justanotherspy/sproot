package main

import (
	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var path string
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a sproot.yaml file",
		Long:  "validate loads a sproot.yaml and reports any missing or invalid fields. Defaults to the sproot_config_path from ~/.sproot/config.yaml, or sproot.yaml in the current directory. A missing sproot.yaml is a warning by default; use --strict to treat it as an error.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path == "" {
				path = resolveDefaultSprootPath()
			}
			return host.RunValidateSprootConfig(path, strict)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "path to sproot.yaml (default: sproot_config_path from ~/.sproot/config.yaml, or sproot.yaml)")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat a missing sproot.yaml as an error instead of a warning")
	return cmd
}

// resolveDefaultSprootPath returns the sproot_config_path from the host config if readable,
// falling back to "sproot.yaml" in the current directory.
func resolveDefaultSprootPath() string {
	hostPath, err := config.DefaultHostConfigPath()
	if err != nil {
		return "sproot.yaml"
	}
	hcfg, err := config.LoadHostConfig(hostPath)
	if err != nil || hcfg.SprootConfigPath == "" {
		return "sproot.yaml"
	}
	return hcfg.SprootConfigPath
}
