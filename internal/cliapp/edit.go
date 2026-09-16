package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/serverstore"
)

func newEditCmd() *cobra.Command {
	var (
		serverType string
		version    string
		confirm    bool
	)

	cmd := &cobra.Command{
		Use:   "edit <name>",
		Short: "Change a server's software (type) and/or Minecraft version",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			opts := fleet.EditOptions{Type: serverType, Version: version}

			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			meta, err := serverstore.Load(f.Root, name)
			if err != nil {
				return err
			}

			if opts.TypeOrVersionChanging(meta) && !confirm {
				return fmt.Errorf(
					"changing type/version recreates %q's container and may break installed mods/plugins that don't yet support the new target — re-run with --confirm once you've read this",
					name,
				)
			}

			if err := f.Edit(cmd.Context(), name, opts); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated %q\n", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverType, "type", "", "new server software (e.g. PAPER)")
	cmd.Flags().StringVar(&version, "version", "", "new Minecraft version")
	cmd.Flags().BoolVar(&confirm, "confirm", false, "confirm a type/version change that may break mods/plugins")

	return cmd
}
