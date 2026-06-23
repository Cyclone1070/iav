// Package cmd implements the CLI interface for the iav test runner using cobra.
package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os/signal"
	"syscall"

	"github.com/Cyclone1070/iav/internal/docker"
	"github.com/Cyclone1070/iav/internal/domain"
	iavfs "github.com/Cyclone1070/iav/internal/fs"
	"github.com/Cyclone1070/iav/internal/pipeline"
	dockerstages "github.com/Cyclone1070/iav/internal/stages/docker"
	"github.com/Cyclone1070/iav/internal/workflow"
	"github.com/spf13/cobra"
)

type fileSystem interface {
	Open(name string) (fs.File, error)
	Abs(path string) (string, error)
	Stat(path string) (fs.FileInfo, error)
	WalkDir(root string, fn fs.WalkDirFunc) error
	EvalSymlinks(path string) (string, error)
}

var defaultFS fileSystem = iavfs.RealFS{}

type dockerClient interface {
	Close() error
	PruneExpiredResources(ctx context.Context) error
	ComposeUp(ctx context.Context, composeFiles []string, runID string) error
	ComposeDown(ctx context.Context, composeFiles []string, runID string) error
	RunContainerInfo(ctx context.Context, opts domain.RunOptions) (stdout string, stderr string, exitCode int, err error)
}

var dockerClientFactory = func() (dockerClient, error) {
	return docker.NewDockerClient()
}

type pipelineAdapter struct {
	*pipeline.Pipeline
}

func (a pipelineAdapter) Add(s workflow.Stage) {
	a.Pipeline.Add(s)
}

func NewTestCmd() *cobra.Command {
	var timeout int

	testCmd := &cobra.Command{
		Use:   "test <script_or_directory>",
		Short: "Run IaV tests for a script or all scripts in a directory",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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

			discoverer := workflow.NewDiscovery(defaultFS)

			composeStages := []workflow.Stage{
				dockerstages.NewLintStage(client),
				dockerstages.NewComposeUpStage(client, defaultFS),
				dockerstages.NewComposeDownStage(client, defaultFS),
			}

			pipelineFactory := func() workflow.Pipeline {
				return pipelineAdapter{Pipeline: pipeline.NewPipeline()}
			}

			composeRunner := workflow.NewRunCompose(defaultFS, client, composeStages, pipelineFactory)

			engine := workflow.NewEngine(discoverer, client, composeRunner, cmd.OutOrStdout())
			return engine.Run(ctx, args[0], timeout)
		},
	}

	testCmd.Flags().IntVar(&timeout, "timeout", 30000, "Timeout in milliseconds")

	return testCmd
}
