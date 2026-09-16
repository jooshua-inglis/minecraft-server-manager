package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

func newCreateCmd() *cobra.Command {
	var (
		serverType string
		version    string
		memory     string
		port       int
		acceptEULA bool
	)

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new server (does not start it)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			f, err := newFleet()
			if err != nil {
				return err
			}
			defer f.Docker.Close()

			meta, err := f.Create(cmd.Context(), args[0], fleet.CreateOptions{
				Type:       serverType,
				Version:    version,
				Memory:     memory,
				Port:       port,
				AcceptEULA: acceptEULA,
			})
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "created %q (type=%s version=%s port=%d)\n",
				meta.Name, meta.Type, meta.Version, meta.Port)
			fmt.Fprintf(cmd.OutOrStdout(), "run `mcm start %s` to boot it\n", meta.Name)
			return nil
		},
	}

	cmd.Flags().StringVar(&serverType, "type", "VANILLA", "server software (VANILLA, PAPER, SPIGOT, FORGE, FABRIC, ...)")
	cmd.Flags().StringVar(&version, "version", "LATEST", "Minecraft version")
	cmd.Flags().StringVar(&memory, "memory", "2G", "JVM heap size")
	cmd.Flags().IntVar(&port, "port", 0, "host port for the game server (0 = auto-pick a free port)")
	cmd.Flags().BoolVar(&acceptEULA, "accept-eula", false, "confirm you have read and accept the Minecraft EULA (https://www.minecraft.net/en-us/eula)")

	return cmd
}
