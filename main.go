// Command iav runs integration acceptance tests for Docker/Compose workflows.
package main

import (
	"fmt"
	"os"

	"github.com/Cyclone1070/iav/cmd"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "iav",
		Short: "Infrastructure-as-Vibe (iav) Local CLI & MCP Test Runner",
	}

	rootCmd.AddCommand(cmd.NewTestCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
