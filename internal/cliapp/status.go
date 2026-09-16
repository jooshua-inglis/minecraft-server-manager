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
			return nil
		},
	}
}
