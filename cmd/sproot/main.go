package main

import (
	"os"

	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=vX.Y.Z".
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "sproot",
		Short:   "sproot bootstraps sprite.dev sprites from a config repo",
		Long:    "sproot reads a sproot.yaml from your config repo and runs each phase to provision a sprite environment.",
		Version: version,
	}
	root.AddCommand(newSetupCmd())
	return root
}
