// Package cmd implements the CLI interface for the iav test runner using cobra.
package cmd

import (
	"context"
	"fmt"
	"io/fs"
	"os/signal"
	"path/filepath"
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

func detectWorkflows(target string) (hasLint bool, hasTest bool, err error) {
	dockerfile := filepath.Join(target, "Dockerfile")
	_, lintErr := defaultFS.Stat(dockerfile)
	hasLint = lintErr == nil

	// Check test compose scripts
	discoverer := workflow.NewDiscovery(defaultFS)
	scripts, _ := discoverer.Run(target)
	hasTest = len(scripts) > 0

	if !hasLint && !hasTest {
		return false, false, fmt.Errorf("no Dockerfile or test compose files found in %q", target)
	}
	return hasLint, hasTest, nil
}

func runRootWorkflow(cmd *cobra.Command, target string, timeout int) error {
	hasLint, hasTest, detectErr := detectWorkflows(target)
	if detectErr != nil {
		return detectErr
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

	if hasLint {
		p := pipelineAdapter{Pipeline: pipeline.NewPipeline(cmd.OutOrStdout())}
		s := dockerstages.NewLintStage(client)
		linter := workflow.NewDockerLinter(p, s, cmd.OutOrStdout())
		if lintErr := linter.Run(ctx, target); lintErr != nil {
			return lintErr
		}
	}

	if hasTest {
		discoverer := workflow.NewDiscovery(defaultFS)
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
		if testErr := tester.Run(ctx, target, timeout); testErr != nil {
			return testErr
		}
	}

	return nil
}

func NewRootCmd() *cobra.Command {
	var timeout int

	rootCmd := &cobra.Command{
		Use:   "iav [target_directory]",
		Short: "Infrastructure-as-Vibe (iav) Local CLI & MCP Test Runner",
		Long:  "Executes linting if a Dockerfile is present, and testing if compose files exist.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := "."
			if len(args) > 0 {
				target = args[0]
			}
			return runRootWorkflow(cmd, target, timeout)
		},
	}

	rootCmd.Flags().IntVar(&timeout, "timeout", 30000, "Timeout in milliseconds")

	rootCmd.AddCommand(NewTestCmd())
	rootCmd.AddCommand(NewLintCmd())

	return rootCmd
}
