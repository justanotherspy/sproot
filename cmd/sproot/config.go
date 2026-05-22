package main

import (
	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the host config file (~/.sproot/config.yaml)",
	}
	cmd.AddCommand(newConfigInitCmd(), newConfigValidateCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var nonInteractive bool
	var source string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a skeleton ~/.sproot/config.yaml",
		Long: "init writes a host config to ~/.sproot/config.yaml. " +
			"By default it prompts for each field interactively, starting with config_source (git or local). " +
			"Use --non-interactive to write a skeleton file with placeholder values. " +
			"Use --source to set the config_source in the skeleton (default: git).",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultHostConfigPath()
			if err != nil {
				return err
			}
			if nonInteractive {
				return host.RunConfigInit(path, source)
			}
			return host.RunConfigInitInteractive(path)
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "write a skeleton file with placeholder values instead of prompting")
	cmd.Flags().StringVar(&source, "source", "git", "config source for the skeleton: git or local (only with --non-interactive)")
	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate ~/.sproot/config.yaml",
		Long:  "validate loads ~/.sproot/config.yaml and reports any missing or invalid fields.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultHostConfigPath()
			if err != nil {
				return err
			}
			return host.RunConfigValidate(path)
		},
	}
}
