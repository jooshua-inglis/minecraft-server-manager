package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status <name>",
		Short: "Show a single server's detailed status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			st, err := f.Status(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "name:       %s\n", st.Metadata.Name)
			fmt.Fprintf(out, "type:       %s\n", st.Metadata.Type)
			fmt.Fprintf(out, "version:    %s\n", st.Metadata.Version)
			fmt.Fprintf(out, "port:       %d\n", st.Metadata.Port)
			fmt.Fprintf(out, "rcon port:  127.0.0.1:%d\n", st.Metadata.RCONPort)
			fmt.Fprintf(out, "data dir:   %s\n", st.DataDir)
			fmt.Fprintf(out, "created:    %s\n", st.Metadata.CreatedAt.Format("2006-01-02 15:04:05 MST"))

			if st.Info == nil {
				fmt.Fprintln(out, "container:  not created")
				return nil
			}

			fmt.Fprintf(out, "container:  %s\n", st.Metadata.ContainerName)
			if st.Info.State != nil {
				fmt.Fprintf(out, "status:     %s\n", st.Info.State.Status)
				if st.Info.State.Running {
					fmt.Fprintf(out, "started at: %s\n", st.Info.State.StartedAt)
				}
				if st.Info.State.Error != "" {
					fmt.Fprintf(out, "last error: %s\n", st.Info.State.Error)
				}
			}

			if st.Crash != nil {
				reason := "OOM killed"
				if !st.Crash.OOMKilled {
					reason = fmt.Sprintf("exit code %d", st.Crash.ExitCode)
				}
				if st.Crash.GaveUp {
					fmt.Fprintf(out, "crash:      gave up after %d/%d restarts (%s) — fix the problem below, then `mcm start %s`\n",
						st.Crash.RestartCount, st.Crash.MaxRetries, reason, st.Metadata.Name)
				} else {
					fmt.Fprintf(out, "crash:      restarting (attempt %d/%d, %s)\n",
						st.Crash.RestartCount, st.Crash.MaxRetries, reason)
				}
				if len(st.Crash.LastLogLines) > 0 {
					fmt.Fprintln(out, "last log lines:")
					for _, line := range st.Crash.LastLogLines {
						fmt.Fprintf(out, "  %s\n", line)
					}
				}
			}
			return nil
		},
	}
}
