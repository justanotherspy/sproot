package main

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/justanotherspy/sproot/internal/host"
	"github.com/justanotherspy/sproot/pkg/log"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	var debug bool

	root := &cobra.Command{
		Use:     "sproot",
		Short:   "sproot bootstraps sprite.dev sprites from a config repo",
		Long:    "sproot reads a sproot.yaml from your config repo and runs each phase to provision a sprite environment.",
		Version: version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			log.SetDebug(debug)
			return nil
		},
		PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
			// Notify (at most once a day) that a newer release exists. Skipped for
			// "setup" (runs in-sprite) and "self-update" (does its own upstream check).
			switch cmd.Name() {
			case "setup", "self-update":
			default:
				host.NotifyUpdateAvailable(version)
			}
			return nil
		},
	}
	root.PersistentFlags().BoolVar(&debug, "debug", false, "enable verbose debug logging")

	root.AddCommand(
		newSetupCmd(),
		newNewCmd(),
		newDestroyCmd(),
		newSpriteStatusCmd(),
		newConfigCmd(),
		newValidateCmd(),
		newConsoleCmd(),
		newListCmd(),
		newExecCmd(),
		newUpgradeCmd(),
		newSelfUpdateCmd(),
		newCheckpointCmd(),
		newCheckpointsCmd(),
		newRestoreCmd(),
		newPushCmd(),
		newOutdatedCmd(),
	)
	return root
}
