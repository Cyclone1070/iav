package orchestrator

import (
	"context"
	"sync"
	"testing"

	"github.com/Cyclone1070/iav/internal/orchestrator/models"
	"github.com/Cyclone1070/iav/internal/ui"
	uimodels "github.com/Cyclone1070/iav/internal/ui/models"
)

func TestCheckShell_Concurrency(t *testing.T) {
	policy := &models.Policy{
		Shell: models.ShellPolicy{
			Allow:        []string{},
			Deny:         []string{},
			SessionAllow: make(map[string]bool),
		},
	}

	mockUI := &MockUI{
		ReadPermissionFunc: func(ctx context.Context, prompt string, preview *uimodels.ToolPreview) (ui.PermissionDecision, error) {
			return ui.DecisionAllowAlways, nil
		},
	}

	ps := NewPolicyService(policy, mockUI)

	// Spawn multiple goroutines requesting permission for different commands
	var wg sync.WaitGroup
	commands := []string{"docker", "git", "npm", "go", "python"}

	for _, cmd := range commands {
		wg.Add(1)
		go func(command string) {
			defer wg.Done()
			err := ps.CheckShell(context.Background(), []string{command})
			if err != nil {
				t.Errorf("CheckShell failed for %s: %v", command, err)
			}
		}(cmd)
	}

	wg.Wait()

	// Verify all commands were added to SessionAllow
	for _, cmd := range commands {
		if !policy.Shell.SessionAllow[cmd] {
			t.Errorf("Command %s not found in SessionAllow", cmd)
		}
	}

	// Verify no commands were lost (should have exactly 5)
	if len(policy.Shell.SessionAllow) != len(commands) {
		t.Errorf("Expected %d commands in SessionAllow, got %d", len(commands), len(policy.Shell.SessionAllow))
	}
}

func TestCheckTool_Concurrency(t *testing.T) {
	policy := &models.Policy{
		Tools: models.ToolPolicy{
			Allow:        []string{},
			Deny:         []string{},
			SessionAllow: make(map[string]bool),
		},
	}

	mockUI := &MockUI{
		ReadPermissionFunc: func(ctx context.Context, prompt string, preview *uimodels.ToolPreview) (ui.PermissionDecision, error) {
			return ui.DecisionAllowAlways, nil
		},
	}

	ps := NewPolicyService(policy, mockUI)

	// Spawn multiple goroutines requesting permission for different tools
	var wg sync.WaitGroup
	tools := []string{"read_file", "write_file", "edit_file", "shell", "find_file"}

	for _, tool := range tools {
		wg.Add(1)
		go func(toolName string) {
			defer wg.Done()
			err := ps.CheckTool(context.Background(), toolName, nil)
			if err != nil {
				t.Errorf("CheckTool failed for %s: %v", toolName, err)
			}
		}(tool)
	}

	wg.Wait()

	// Verify all tools were added to SessionAllow
	for _, tool := range tools {
		if !policy.Tools.SessionAllow[tool] {
			t.Errorf("Tool %s not found in SessionAllow", tool)
		}
	}

	// Verify no tools were lost (should have exactly 5)
	if len(policy.Tools.SessionAllow) != len(tools) {
		t.Errorf("Expected %d tools in SessionAllow, got %d", len(tools), len(policy.Tools.SessionAllow))
	}
}
