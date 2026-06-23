package workflow

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Cyclone1070/iav/internal/domain"
)

type DiscoveryService interface {
	Run(targetPath string) ([]string, error)
}

type Pruner interface {
	PruneExpiredResources(ctx context.Context) error
}

type Runner interface {
	Run(ctx context.Context, testComposePath string, runID string, timeoutMs int) (*domain.TestExecutionResult, error)
}

type Engine struct {
	discoverer DiscoveryService
	pruner     Pruner
	runner     Runner
	out        io.Writer
}

func NewEngine(
	discoverer DiscoveryService,
	pruner Pruner,
	runner Runner,
	out io.Writer,
) *Engine {
	return &Engine{
		discoverer: discoverer,
		pruner:     pruner,
		runner:     runner,
		out:        out,
	}
}

func (e *Engine) Run(ctx context.Context, targetPath string, timeoutMs int) error {
	testScripts, err := e.discoverer.Run(targetPath)
	if err != nil {
		return err
	}

	_ = e.pruner.PruneExpiredResources(ctx)

	var failedScripts []string
	for _, scriptPath := range testScripts {
		if err := e.executeSingleTest(ctx, scriptPath, timeoutMs); err != nil {
			failedScripts = append(failedScripts, scriptPath)
		}
	}

	if len(failedScripts) > 0 {
		return fmt.Errorf("the following test scripts failed: %s", strings.Join(failedScripts, ", "))
	}

	return nil
}

func (e *Engine) executeSingleTest(ctx context.Context, testComposePath string, timeoutMs int) error {
	_, _ = fmt.Fprintf(e.out, "Running test compose: %s\n", filepath.Base(testComposePath))

	runID := e.generateRunID()
	result, pipelineErr := e.runner.Run(ctx, testComposePath, runID, timeoutMs)

	if result != nil {
		e.printResultLog(testComposePath, result)
	}

	return pipelineErr
}

func (e *Engine) generateRunID() string {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return strconv.FormatInt(time.Now().UnixNano(), 10)
	}
	return hex.EncodeToString(bytes)
}

func (e *Engine) printResultLog(scriptPath string, result *domain.TestExecutionResult) {
	_, _ = fmt.Fprintln(e.out)
	statusStr := "PASSED"
	if !result.Success {
		statusStr = "FAILED"
	}
	_, _ = fmt.Fprintf(e.out, "Test Execution Summary for %s: %s\n", filepath.Base(scriptPath), statusStr)
	_, _ = fmt.Fprintln(e.out)
}
