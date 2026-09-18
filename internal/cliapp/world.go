package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newWorldCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "world",
		Short: "Manage a server's world save, independent of its type/version/mods",
		Long: "Complements `mcm backup` (M10), which snapshots a server's entire\n" +
			"data/ directory. `mcm world` scopes to just the world save itself —\n" +
			"back it up, reset it, or move it between servers (even ones running\n" +
			"different server software) without touching mods, plugins, configs,\n" +
			"or the whitelist. The server must be stopped for every subcommand.",
	}

	backup := &cobra.Command{
		Use:   "backup <server>",
		Short: "Snapshot just a server's world save",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			info, err := f.WorldBackup(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "backed up %q's world as %q (%s)\n", args[0], info.ID, humanBytes(info.SizeBytes))
			return nil
		},
	}
	backupList := &cobra.Command{
		Use:   "list <server>",
		Short: "List a server's world backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			backups, err := f.WorldBackupList(args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(backups) == 0 {
				fmt.Fprintln(out, "no world backups")
				return nil
			}
			tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "ID\tCREATED\tSIZE")
			for _, b := range backups {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", b.ID, b.CreatedAt.Format("2006-01-02 15:04:05"), humanBytes(b.SizeBytes))
			}
			return tw.Flush()
		},
	}
	backup.AddCommand(backupList)

	restore := &cobra.Command{
		Use:   "restore <server> <backup-id>",
		Short: "Replace a server's world save with a previously taken world backup",
		Long:  "See `mcm world backup list <server>` for available backup IDs.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WorldRestore(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restored %q's world from backup %q\n", args[0], args[1])
			return nil
		},
	}

	reset := &cobra.Command{
		Use:   "reset <server>",
		Short: "Delete a server's world so it generates a fresh one on next start",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WorldReset(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "reset %q's world — it'll generate a new one on next start\n", args[0])
			return nil
		},
	}

	export := &cobra.Command{
		Use:   "export <server> <output.tar.gz>",
		Short: "Export a server's world save as a portable archive",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WorldExport(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "exported %q's world to %q\n", args[0], args[1])
			return nil
		},
	}

	importCmd := &cobra.Command{
		Use:   "import <server> <input.tar.gz>",
		Short: "Replace a server's world save with a previously exported archive",
		Long:  "Works across servers, even ones running different server software\n" + "or Minecraft versions than whatever produced the export.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.WorldImport(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "imported %q into %q's world\n", args[1], args[0])
			return nil
		},
	}

	cmd.AddCommand(backup, restore, reset, export, importCmd)
	return cmd
}
