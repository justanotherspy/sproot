package main

import (
	"github.com/justanotherspy/sproot/internal/config"
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the host config file (~/.sproot/config)",
	}
	cmd.AddCommand(newConfigInitCmd(), newConfigValidateCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a skeleton ~/.sproot/config",
		Long: "init writes a host config to ~/.sproot/config. " +
			"By default it prompts for each field interactively. " +
			"Use --non-interactive to write a skeleton file with placeholder values instead.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultHostConfigPath()
			if err != nil {
				return err
			}
			if nonInteractive {
				return host.RunConfigInit(path)
			}
			return host.RunConfigInitInteractive(path)
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "write a skeleton file with placeholder values instead of prompting")
	return cmd
}

func newConfigValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate ~/.sproot/config",
		Long:  "validate loads ~/.sproot/config and reports any missing or invalid fields.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultHostConfigPath()
			if err != nil {
				return err
			}
			return host.RunConfigValidate(path)
		},
	}
}
