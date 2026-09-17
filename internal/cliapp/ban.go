package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newBanCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ban",
		Short: "Manage a server's banned players",
	}

	var offline bool
	var reason string
	add := &cobra.Command{
		Use:   "add <server> <player>",
		Short: "Ban a player",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.BanAdd(cmd.Context(), args[0], args[1], reason, offline); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "banned %q from %q\n", args[1], args[0])
			return nil
		},
	}
	add.Flags().BoolVar(&offline, "offline", false, "resolve the player's UUID using the offline-mode algorithm instead of looking it up from Mojang (use this if the server runs with online-mode=false)")
	add.Flags().StringVar(&reason, "reason", "", "ban reason (default: \"Banned by an operator.\")")

	remove := &cobra.Command{
		Use:   "remove <server> <player>",
		Short: "Pardon (unban) a player",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.BanRemove(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "pardoned %q on %q\n", args[1], args[0])
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list <server>",
		Short: "List banned players",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			entries, err := f.BanList(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no banned players")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tREASON\tEXPIRES")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\t%s\n", e.Name, e.Reason, e.Expires)
			}
			return w.Flush()
		},
	}

	cmd.AddCommand(add, remove, list)
	return cmd
}
