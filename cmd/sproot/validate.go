package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newValidateCmd() *cobra.Command {
	var path string
	var strict bool

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate a sproot.yaml file",
		Long:  "validate loads a sproot.yaml and reports any missing or invalid fields. With no --path, it resolves the sproot.yaml from the configured source in ~/.sproot/config.yaml (the git config repo, or the local config directory) so it checks the same file `sproot new` would use. Pass --path to validate a specific local file instead. A missing sproot.yaml is a warning by default; use --strict to treat it as an error.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if path != "" {
				return host.RunValidateSprootConfig(path, strict)
			}
			return host.RunValidate(strict)
		},
	}

	cmd.Flags().StringVar(&path, "path", "", "validate a specific local sproot.yaml instead of the configured config source")
	cmd.Flags().BoolVar(&strict, "strict", false, "treat a missing sproot.yaml as an error instead of a warning")
	return cmd
}
