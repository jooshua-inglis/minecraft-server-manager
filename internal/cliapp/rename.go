package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old-name> <new-name>",
		Short: "Rename a server",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if err := f.Rename(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "renamed %q to %q\n", args[0], args[1])
			return nil
		},
	}
}
