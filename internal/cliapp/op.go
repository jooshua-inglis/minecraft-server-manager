package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newOpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "op",
		Short: "Manage a server's operators",
	}

	var offline bool
	add := &cobra.Command{
		Use:   "add <server> <player>",
		Short: "Grant a player operator status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.OpAdd(cmd.Context(), args[0], args[1], offline); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "opped %q on %q\n", args[1], args[0])
			return nil
		},
	}
	add.Flags().BoolVar(&offline, "offline", false, "resolve the player's UUID using the offline-mode algorithm instead of looking it up from Mojang (use this if the server runs with online-mode=false)")

	remove := &cobra.Command{
		Use:   "remove <server> <player>",
		Short: "Revoke a player's operator status",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.OpRemove(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "deopped %q on %q\n", args[1], args[0])
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list <server>",
		Short: "List operators",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			entries, err := f.OpList(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no operators")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tLEVEL\tUUID")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%d\t%s\n", e.Name, e.Level, e.UUID)
			}
			return w.Flush()
		},
	}

	cmd.AddCommand(add, remove, list)
	return cmd
}
