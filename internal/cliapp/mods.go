package cliapp

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

func newModsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mods",
		Short: "Manage a server's individually installed mods/plugins",
	}

	var addSource string
	add := &cobra.Command{
		Use:   "add <server> <ref>",
		Short: "Install a mod or plugin and recreate the server's container",
		Long: "Install a mod or plugin. <ref> is a Modrinth project slug/ID/URL, a\n" +
			"CurseForge project:file reference, or (with --source url) a direct\n" +
			"download URL. Recreates the server's container so it takes effect;\n" +
			"restarts it afterward if it was already running.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.ModsAdd(cmd.Context(), args[0], addSource, args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "added %q to %q (%s)\n", args[1], args[0], addSource)
			return nil
		},
	}
	add.Flags().StringVar(&addSource, "source", fleet.SourceModrinth, "where to install from: modrinth, curseforge, or url")

	var removeSource string
	remove := &cobra.Command{
		Use:   "remove <server> <ref>",
		Short: "Remove a mod or plugin and recreate the server's container",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			if err := f.ModsRemove(cmd.Context(), args[0], removeSource, args[1]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed %q from %q (%s)\n", args[1], args[0], removeSource)
			return nil
		},
	}
	remove.Flags().StringVar(&removeSource, "source", fleet.SourceModrinth, "which list to remove from: modrinth, curseforge, or url")

	list := &cobra.Command{
		Use:   "list <server>",
		Short: "List a server's individually installed mods/plugins",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()
			mods, err := f.ModsList(cmd.Context(), args[0])
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			printGroup(out, "modrinth", mods.ModrinthProjects)
			printGroup(out, "curseforge", mods.CurseForgeFiles)
			printGroup(out, "mods (url)", mods.ModURLs)
			printGroup(out, "plugins (url)", mods.PluginURLs)
			return nil
		},
	}

	cmd.AddCommand(add, remove, list)
	return cmd
}

func printGroup(out io.Writer, label string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintf(out, "%s:\n", label)
	for _, item := range items {
		fmt.Fprintf(out, "  %s\n", item)
	}
}
