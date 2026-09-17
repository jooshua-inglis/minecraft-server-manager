package cliapp

import (
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var follow bool
	var tail string

	cmd := &cobra.Command{
		Use:   "logs <name>",
		Short: "Show (optionally follow) a server's console output",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			return f.Logs(cmd.Context(), args[0], follow, tail, cmd.OutOrStdout())
		},
	}

	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "stream new log lines as they're written")
	cmd.Flags().StringVar(&tail, "tail", "all", "number of lines to show from the end of the logs, or \"all\"")

	return cmd
}
