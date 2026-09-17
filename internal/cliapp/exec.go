package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newExecCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "exec <name> <command>",
		Short: "Run a console command on a running server via RCON",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			name := args[0]
			command := args[1]
			for _, arg := range args[2:] {
				command += " " + arg
			}

			output, err := f.Exec(cmd.Context(), name, command)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), output)
			return nil
		},
	}
}
