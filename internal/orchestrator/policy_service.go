package orchestrator

import (
	"context"
	"fmt"
	"path/filepath"
	"slices"
	"sync"

	"github.com/Cyclone1070/deployforme/internal/orchestrator/models"
	"github.com/Cyclone1070/deployforme/internal/ui"
)

// policyService implements models.PolicyService
type policyService struct {
	policy *models.Policy
	ui     ui.UserInterface
	mu     sync.RWMutex // Protects SessionAllow maps
}

// NewPolicyService creates a new PolicyService instance
func NewPolicyService(policy *models.Policy, userInterface ui.UserInterface) models.PolicyService {
	return &policyService{
		policy: policy,
		ui:     userInterface,
	}
}

// CheckShell validates if a shell command is allowed to execute
func (p *policyService) CheckShell(ctx context.Context, command []string) error {
	if len(command) == 0 {
		return fmt.Errorf("command cannot be empty")
	}

	root := getCommandRoot(command)
	if root == "" {
		return fmt.Errorf("invalid command")
	}

	// Use single Lock for entire check-and-update operation to prevent races
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Check SessionAllow (Override)
	if p.policy.Shell.SessionAllow != nil && p.policy.Shell.SessionAllow[root] {
		return nil
	}

	// 2. Check Allow List
	if slices.Contains(p.policy.Shell.Allow, root) {
		return nil
	}

	// 3. Check Deny List
	if slices.Contains(p.policy.Shell.Deny, root) {
		return fmt.Errorf("command '%s' is denied by policy", root)
	}

	// 4. Ask user for permission
	// Unlock before blocking I/O operation
	p.mu.Unlock()
	prompt := fmt.Sprintf("Agent wants to execute shell command: %s\nAllow this command?", root)
	decision, err := p.ui.ReadPermission(ctx, prompt)
	p.mu.Lock() // Re-acquire lock before updating map
	if err != nil {
		return fmt.Errorf("failed to get user permission: %w", err)
	}

	switch decision {
	case ui.DecisionAllow:
		return nil
	case ui.DecisionDeny:
		return fmt.Errorf("user denied command '%s'", root)
	case ui.DecisionAllowAlways:
		// Update SessionAllow
		if p.policy.Shell.SessionAllow == nil {
			p.policy.Shell.SessionAllow = make(map[string]bool)
		}
		p.policy.Shell.SessionAllow[root] = true
		return nil
	default:
		return fmt.Errorf("invalid permission decision: %s", decision)
	}
}

// CheckTool validates if a tool is allowed to be used
func (p *policyService) CheckTool(ctx context.Context, toolName string) error {
	if toolName == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Use single Lock for entire check-and-update operation to prevent races
	p.mu.Lock()
	defer p.mu.Unlock()

	// 1. Check SessionAllow (Override)
	if p.policy.Tools.SessionAllow != nil && p.policy.Tools.SessionAllow[toolName] {
		return nil
	}

	// 2. Check Allow List
	if slices.Contains(p.policy.Tools.Allow, toolName) {
		return nil
	}

	// 3. Check Deny List
	if slices.Contains(p.policy.Tools.Deny, toolName) {
		return fmt.Errorf("tool '%s' is denied by policy", toolName)
	}

	// 4. Ask user for permission
	// Unlock before blocking I/O operation
	p.mu.Unlock()
	prompt := fmt.Sprintf("Agent wants to use tool: %s\nAllow this tool?", toolName)
	decision, err := p.ui.ReadPermission(ctx, prompt)
	p.mu.Lock() // Re-acquire lock before updating map
	if err != nil {
		return fmt.Errorf("failed to get user permission: %w", err)
	}

	switch decision {
	case ui.DecisionAllow:
		return nil
	case ui.DecisionDeny:
		return fmt.Errorf("user denied tool '%s'", toolName)
	case ui.DecisionAllowAlways:
		// Update SessionAllow
		if p.policy.Tools.SessionAllow == nil {
			p.policy.Tools.SessionAllow = make(map[string]bool)
		}
		p.policy.Tools.SessionAllow[toolName] = true
		return nil
	default:
		return fmt.Errorf("invalid permission decision: %s", decision)
	}
}

// getCommandRoot extracts the root command (basename) from a command slice.
// It handles full paths by extracting just the command name.
// Example: ["/usr/bin/docker", "run"] returns "docker".
func getCommandRoot(command []string) string {
	if len(command) == 0 {
		return ""
	}
	// Handle paths like /usr/bin/docker
	return filepath.Base(command[0])
}
