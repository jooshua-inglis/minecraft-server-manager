package cliapp

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

const watchInterval = 2 * time.Second

func newTopCmd() *cobra.Command {
	var watch bool

	cmd := &cobra.Command{
		Use:   "top",
		Short: "Show CPU/memory usage and player counts across the fleet",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			if !watch {
				return printTop(cmd, f)
			}

			for {
				fmt.Fprint(cmd.OutOrStdout(), "\033[H\033[2J")
				if err := printTop(cmd, f); err != nil {
					return err
				}
				time.Sleep(watchInterval)
			}
		},
	}

	cmd.Flags().BoolVarP(&watch, "watch", "w", false, "refresh every 2 seconds until interrupted")

	return cmd
}

func printTop(cmd *cobra.Command, f *fleet.Fleet) error {
	views, err := f.Top(cmd.Context())
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if len(views) == 0 {
		fmt.Fprintln(out, "no servers yet — create one with `mcm create <name>`")
		return nil
	}

	w := tabwriter.NewWriter(out, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSTATUS\tCPU%\tMEMORY\tPLAYERS")
	for _, v := range views {
		mem := "-"
		if v.MemLimitBytes > 0 {
			mem = fmt.Sprintf("%.0fMiB / %.0fMiB", mib(v.MemUsageBytes), mib(v.MemLimitBytes))
		}
		cpu := "-"
		if v.Status == "running" {
			cpu = fmt.Sprintf("%.1f%%", v.CPUPercent)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n", v.Name, v.Status, cpu, mem, v.Players)
	}
	return w.Flush()
}

func mib(bytes uint64) float64 {
	return float64(bytes) / 1024 / 1024
}
