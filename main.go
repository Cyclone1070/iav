// Command iav runs integration acceptance tests for Docker/Compose workflows.
package main

import (
	"fmt"
	"os"

	"github.com/Cyclone1070/iav/cmd"
)

func main() {
	rootCmd := cmd.NewRootCmd()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
