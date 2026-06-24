package cmd

import (
	"fmt"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/Cyclone1070/iav/internal/pipeline"
	dockerstages "github.com/Cyclone1070/iav/internal/stages/docker"
	"github.com/Cyclone1070/iav/internal/workflow"
	"github.com/spf13/cobra"
)

func NewLintCmd() *cobra.Command {
	var timeout int

	lintCmd := &cobra.Command{
		Use:   "lint [target_directory]",
		Short: "Lint Dockerfile (no testing)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}

			dockerfile := filepath.Join(target, "Dockerfile")
			if _, err := defaultFS.Stat(dockerfile); err != nil {
				return fmt.Errorf("dockerfile not found in %q", target)
			}

			completed := false
			ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer func() {
				completed = true
				stop()
			}()

			// Print termination signal notification in background
			go func() {
				<-ctx.Done()
				if !completed && ctx.Err() != nil {
					cmd.Println("\nReceived termination signal. Cleaning up resources...")
				}
			}()

			client, err := dockerClientFactory()
			if err != nil {
				return fmt.Errorf("failed to initialize Docker client: %w", err)
			}
			defer func() { _ = client.Close() }()

			p := pipelineAdapter{Pipeline: pipeline.NewPipeline(cmd.OutOrStdout())}
			s := dockerstages.NewLintStage(client)
			linter := workflow.NewDockerLinter(p, s)
			return linter.Run(ctx, target)
		},
	}

	lintCmd.Flags().IntVar(&timeout, "timeout", 30000, "Timeout in milliseconds")

	return lintCmd
}

