package main

import (
	"fmt"
	"os"

	"github.com/jooshua-inglis/minecraft-server-manager/internal/cliapp"
)

func main() {
	if err := cliapp.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
