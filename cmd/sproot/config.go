package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/justanotherspy/sproot/internal/config"
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
	return &cobra.Command{
		Use:   "init",
		Short: "Write a skeleton ~/.sproot/config",
		Long:  "init writes a skeleton host config to ~/.sproot/config with placeholder values. Returns an error if the file already exists.",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.DefaultHostConfigPath()
			if err != nil {
				return err
			}
			return host.RunConfigInit(path)
		},
	}
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
