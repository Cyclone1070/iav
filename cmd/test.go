package cmd

import (
	"fmt"
	"os/signal"
	"syscall"

	"github.com/Cyclone1070/iav/internal/pipeline"
	dockerstages "github.com/Cyclone1070/iav/internal/stages/docker"
	"github.com/Cyclone1070/iav/internal/workflow"
	"github.com/spf13/cobra"
)

func NewTestCmd() *cobra.Command {
	var timeout int

	testCmd := &cobra.Command{
		Use:   "test [target_directory]",
		Short: "Run compose tests (no linting)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}

			// Verify compose files exist to error early
			discoverer := workflow.NewDiscovery(defaultFS)
			scripts, err := discoverer.Run(target)
			if err != nil || len(scripts) == 0 {
				if err != nil {
					return err
				}
				return fmt.Errorf("no valid test compose files found in %q", target)
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

			pipelineFactory := func() workflow.Pipeline {
				p := pipeline.NewPipeline(cmd.OutOrStdout())
				p.SetPrefix("    ")
				return pipelineAdapter{Pipeline: p}
			}

			testStages := []workflow.Stage{
				dockerstages.NewComposeUpStage(client, defaultFS),
				dockerstages.NewComposeDownStage(client, defaultFS),
			}

			composeRunner := workflow.NewRunCompose(defaultFS, client, testStages, pipelineFactory)
			tester := workflow.NewDockerComposeTester(discoverer, client, composeRunner, cmd.OutOrStdout())
			return tester.Run(ctx, target, timeout)
		},
	}

	testCmd.Flags().IntVar(&timeout, "timeout", 30000, "Timeout in milliseconds")

	return testCmd
}
