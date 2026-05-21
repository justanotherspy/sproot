package main

import (
	"os"

	"github.com/spf13/cobra"

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
		newCheckpointCmd(),
		newCheckpointsCmd(),
		newRestoreCmd(),
		newPushCmd(),
	)
	return root
}
