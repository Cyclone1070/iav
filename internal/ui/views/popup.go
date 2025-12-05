package views

import (
	"fmt"
	"strings"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/charmbracelet/lipgloss"
)

// RenderModelPopup renders the model selection popup
func RenderModelPopup(s models.State) string {
	if !s.ShowModelList || len(s.ModelList) == 0 {
		return ""
	}

	var lines []string
	lines = append(lines, lipgloss.NewStyle().Bold(true).Render("Select Model:"))
	lines = append(lines, "")

	for i, model := range s.ModelList {
		if i == s.ModelListIndex {
			// Highlight selected
			lines = append(lines, lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true).
				Render(fmt.Sprintf("▸ %s", model)))
		} else {
			lines = append(lines, fmt.Sprintf("  %s", model))
		}
	}

	lines = append(lines, "")
	lines = append(lines, lipgloss.NewStyle().Faint(true).Render("↑/↓: Navigate  Enter: Select  Esc: Cancel"))

	content := strings.Join(lines, "\n")
	return PermissionBoxStyle.Render(content)
}
