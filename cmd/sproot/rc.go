package main

import (
	"github.com/justanotherspy/sproot/internal/host"
	"github.com/spf13/cobra"
)

func newRCCmd() *cobra.Command {
	var opts host.RCOptions
	cmd := &cobra.Command{
		Use:   "rc <name>",
		Short: "Start or stop a Claude Code Remote Control session on a sprite",
		Long: "rc registers `claude remote-control` as a sprite-env service so the session is reachable " +
			"from claude.ai/code and the Claude mobile app, and keeps the sprite awake until closed. " +
			"Requires a one-time `claude auth login` on the sprite (run it via `sproot console`); the " +
			"inference tokens sproot forwards are not accepted by Remote Control.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Name = args[0]
			return host.RunRC(cmd.Context(), opts)
		},
	}
	cmd.Flags().BoolVar(&opts.Close, "close", false, "stop the remote-control session and let the sprite pause")
	cmd.Flags().StringVar(&opts.Dir, "dir", "", "working directory for the session (default: home directory)")
	cmd.Flags().StringVar(&opts.Spawn, "spawn", "same-dir", "session mode: same-dir, worktree, or session")
	cmd.Flags().StringVar(&opts.SessionName, "session-name", "", "session title shown in the web/mobile UI (default: sprite name)")
	return cmd
}
