package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List every server in the fleet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			views, err := f.List(cmd.Context())
			if err != nil {
				return err
			}

			if len(views) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no servers yet — create one with `mcm create <name>`")
				return nil
			}

			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tTYPE\tVERSION\tPORT\tSTATUS")
			for _, v := range views {
				fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", v.Name, v.Type, v.Version, v.Port, v.Status)
			}
			return w.Flush()
		},
	}
}
