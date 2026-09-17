package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newWhitelistCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "whitelist",
		Short: "Manage a server's whitelist",
	}

	var offline bool
	add := &cobra.Command{
		Use:   "add <server> <player>",
		Short: "Add a player to the whitelist",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WhitelistAdd(cmd.Context(), args[0], args[1], offline); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "whitelisted %q on %q\n", args[1], args[0])
			return nil
		},
	}
	add.Flags().BoolVar(&offline, "offline", false, "resolve the player's UUID using the offline-mode algorithm instead of looking it up from Mojang (use this if the server runs with online-mode=false)")

	remove := &cobra.Command{
		Use:   "remove <server> <player>",
		Short: "Remove a player from the whitelist",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WhitelistRemove(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %q from %q's whitelist\n", args[1], args[0])
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list <server>",
		Short: "List whitelisted players",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			entries, err := f.WhitelistList(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if len(entries) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no whitelisted players")
				return nil
			}
			w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 2, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tUUID")
			for _, e := range entries {
				fmt.Fprintf(w, "%s\t%s\n", e.Name, e.UUID)
			}
			return w.Flush()
		},
	}

	cmd.AddCommand(add, remove, list)
	return cmd
}
