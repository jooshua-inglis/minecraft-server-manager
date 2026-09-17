package cliapp

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

func newModpackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "modpack",
		Short: "Install or remove a server's modpack",
	}

	var installSource string
	install := &cobra.Command{
		Use:   "install <server> <ref>",
		Short: "Point a server at a Modrinth or CurseForge modpack and recreate its container",
		Long: "Switches the server's software to the given modpack's launcher\n" +
			"(TYPE=MODRINTH or TYPE=AUTO_CURSEFORGE) and recreates the container\n" +
			"so itzg resolves and installs it on next start. <ref> is a Modrinth\n" +
			"project slug/ID/URL, or a CurseForge slug/page URL. CurseForge\n" +
			"modpacks require cf_api_key to be set in mcm's config.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.ModpackInstall(cmd.Context(), args[0], installSource, args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed modpack %q on %q (%s); start it to let itzg resolve and download it\n", args[1], args[0], installSource)
			return nil
		},
	}
	install.Flags().StringVar(&installSource, "source", fleet.ModpackSourceModrinth, "where to install from: modrinth or curseforge")

	remove := &cobra.Command{
		Use:   "remove <server>",
		Short: "Remove a server's modpack and revert it to a plain VANILLA server",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.ModpackRemove(cmd.Context(), args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed modpack from %q; reverted to VANILLA\n", args[0])
			return nil
		},
	}

	status := &cobra.Command{
		Use:   "status <server>",
		Short: "Show a server's installed modpack, if any",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			st, err := f.ModpackStatus(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			if st.Ref == "" {
				fmt.Fprintf(out, "no modpack installed (type: %s)\n", st.Type)
				return nil
			}
			fmt.Fprintf(out, "source: %s\nref:    %s\ntype:   %s\n", st.Source, st.Ref, st.Type)
			return nil
		},
	}

	var searchSource string
	var searchLimit int
	search := &cobra.Command{
		Use:   "search <query>",
		Short: "Search Modrinth or CurseForge for a modpack to install",
		Long: "Prints candidate refs for `mcm modpack install`. Modrinth needs no\n" +
			"API key; CurseForge search requires cf_api_key to be set in mcm's\n" +
			"config.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			results, err := f.ModpackSearch(cmd.Context(), searchSource, args[0], searchLimit)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if len(results) == 0 {
				fmt.Fprintln(out, "no results")
				return nil
			}

			tw := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
			fmt.Fprintln(tw, "REF\tNAME\tDOWNLOADS\tDESCRIPTION")
			for _, r := range results {
				fmt.Fprintf(tw, "%s\t%s\t%d\t%s\n", r.Ref, r.Name, r.Downloads, truncate(r.Description, 70))
			}
			return tw.Flush()
		},
	}
	search.Flags().StringVar(&searchSource, "source", fleet.ModpackSourceModrinth, "where to search: modrinth or curseforge")
	search.Flags().IntVar(&searchLimit, "limit", 10, "maximum number of results")

	cmd.AddCommand(install, remove, status, search)
	return cmd
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}
