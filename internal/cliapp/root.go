// Package cliapp wires up mcm's cobra commands. Every command handler is
// a thin translation from flags to a call into the fleet package — no
// business logic lives here, so the future web server can call the same
// fleet functions directly.
package cliapp

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/config"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/dockerctl"
	"github.com/jooshua-inglis/minecraft-server-manager/internal/fleet"
)

var version = "0.1.0-dev"

func Execute() error {
	root := &cobra.Command{
		Use:           "mcm",
		Short:         "mcm manages Minecraft servers running in Docker",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(
		newVersionCmd(),
		newCreateCmd(),
		newStartCmd(),
		newStopCmd(),
		newDestroyCmd(),
		newListCmd(),
		newStatusCmd(),
		newRenameCmd(),
		newEditCmd(),
		newExecCmd(),
	)

	return root.Execute()
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print mcm's version",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), version)
			return nil
		},
	}
}

// newFleet loads config, connects to Docker, and constructs a fleet.Fleet
// for a command to use. Callers are responsible for closing the returned
// Docker client via fleet.Docker.Close().
func newFleet() (*fleet.Fleet, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	docker, err := dockerctl.New()
	if err != nil {
		return nil, err
	}

	return fleet.New(cfg.ServersRoot, docker), nil
}
