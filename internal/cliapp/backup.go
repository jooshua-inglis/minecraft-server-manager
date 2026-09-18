package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

func newBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup <server>",
		Short: "Snapshot a server's entire data directory",
		Long: "Archives the server's whole data/ directory (world, mods, plugins,\n" +
			"configs, whitelist/ops/bans) to backups/<id>.tar.gz next to it. If the\n" +
			"server is running, saves are paused/flushed over RCON first so the\n" +
			"copy isn't taken mid-write.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			info, err := f.Backup(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "backed up %q as %q (%s)\n", args[0], info.ID, humanBytes(info.SizeBytes))
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list <server>",
		Short: "List a server's backups",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			backups, err := f.ListBackups(args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(backups) == 0 {
				fmt.Fprintln(out, "no backups")
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

	cmd.AddCommand(list)
	return cmd
}

func newRestoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <server> <backup-id>",
		Short: "Replace a server's data directory with a previously taken backup",
		Long:  "The server must be stopped first. See `mcm backup list <server>` for available backup IDs.",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.Restore(cmd.Context(), args[0], args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "restored %q from backup %q\n", args[0], args[1])
			return nil
		},
	}
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}
