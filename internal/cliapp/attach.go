package cliapp

import (
	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/tui"
)

func newAttachCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "attach <name>",
		Short: "Open a full-screen terminal dashboard for one server",
		Long: "Combines `mcm logs -f`, `mcm top`, and `mcm exec` into a single\n" +
			"interactive screen: a live log pane, a stats header (CPU/memory/\n" +
			"players), and a command line for sending console commands over\n" +
			"RCON. Ctrl+C to quit.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			return tui.Attach(cmd.Context(), f, args[0])
		},
	}
}
