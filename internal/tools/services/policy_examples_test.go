package services

import (
	"testing"

	"github.com/Cyclone1070/deployforme/internal/tools/models"
)

// TestPolicyBehaviorExamples demonstrates the policy behavior with examples
func TestPolicyBehaviorExamples(t *testing.T) {
	tests := []struct {
		name        string
		policy      models.CommandPolicy
		command     string
		expectedErr error
		description string
	}{
		{
			name:        "Allowed command executes automatically",
			policy:      models.CommandPolicy{Allow: []string{"git"}},
			command:     "git",
			expectedErr: nil,
			description: "Commands in the allow list execute without approval",
		},
		{
			name:        "Denied command is rejected",
			policy:      models.CommandPolicy{Deny: []string{"rm"}},
			command:     "rm",
			expectedErr: models.ErrShellRejected,
			description: "Commands in the deny list are automatically rejected",
		},
		{
			name:        "Unknown command requires approval (default behavior)",
			policy:      models.CommandPolicy{Allow: []string{"git"}, Deny: []string{"rm"}},
			command:     "kubectl",
			expectedErr: models.ErrShellApprovalRequired,
			description: "Commands not in allow or deny lists require user approval",
		},
		{
			name:        "SessionAllow overrides deny",
			policy:      models.CommandPolicy{Deny: []string{"rm"}, SessionAllow: map[string]bool{"rm": true}},
			command:     "rm",
			expectedErr: nil,
			description: "SessionAllow provides runtime override for denied commands",
		},
		{
			name:        "Empty policy defaults to ask",
			policy:      models.CommandPolicy{},
			command:     "anycommand",
			expectedErr: models.ErrShellApprovalRequired,
			description: "With no configured lists, all commands default to asking",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := EvaluatePolicy(tt.policy, []string{tt.command})
			if err != tt.expectedErr {
				t.Errorf("%s\nEvaluatePolicy(%q) = %v, want %v",
					tt.description, tt.command, err, tt.expectedErr)
			}
		})
	}
}
