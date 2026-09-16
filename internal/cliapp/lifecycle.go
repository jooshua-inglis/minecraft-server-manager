package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

func newStartCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "start <name>",
		Short: "Start a server's container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if err := f.Start(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "started %q\n", args[0])
			return nil
		},
	}
}

func newStopCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stop <name>",
		Short: "Stop a server's container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if err := f.Stop(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "stopped %q\n", args[0])
			return nil
		},
	}
}

func newDestroyCmd() *cobra.Command {
	var purge bool
	var yes bool

	cmd := &cobra.Command{
		Use:   "destroy <name>",
		Short: "Remove a server's container",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if purge && !yes {
				return fmt.Errorf("--purge deletes the server's data directory permanently; re-run with --yes to confirm")
			}

			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if err := f.Destroy(cmd.Context(), args[0], fleet.DestroyOptions{Purge: purge}); err != nil {
				return err
			}

			if purge {
				fmt.Fprintf(cmd.OutOrStdout(), "destroyed %q and deleted its data directory\n", args[0])
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "destroyed %q (data directory kept; pass --purge --yes to delete it too)\n", args[0])
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&purge, "purge", false, "also delete the server's data directory")
	cmd.Flags().BoolVar(&yes, "yes", false, "confirm a destructive --purge")

	return cmd
}
