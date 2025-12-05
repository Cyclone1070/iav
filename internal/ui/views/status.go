package views

import (
	"fmt"
	"strings"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/charmbracelet/lipgloss"
)

// RenderStatus renders the status bar
func RenderStatus(s models.State) string {
	var icon string
	var style lipgloss.Style

	switch s.StatusPhase {
	case "executing":
		icon = s.Spinner.View()
		style = StatusExecutingStyle
	case "done":
		icon = "✔"
		style = StatusDoneStyle
	case "thinking":
		icon = s.Spinner.View()
		style = StatusThinkingStyle
		// Animate the dots
		dots := strings.Repeat(".", s.DotCount)
		return style.Render(fmt.Sprintf("%s Generating%s", icon, dots))
	default:
		style = StatusDefaultStyle
	}

	status := "Ready"
	if s.StatusMessage != "" {
		status = fmt.Sprintf("%s %s", icon, s.StatusMessage)
	} else if s.StatusPhase != "ready" && s.StatusPhase != "" {
		// If thinking/executing but no message, show icon
		status = icon
	}

	leftSide := style.Render(status)

	// Right side: Model name
	rightSide := ""
	if s.CurrentModel != "" {
		rightSide = StatusDefaultStyle.Copy().
			Foreground(lipgloss.Color("241")). // Dim gray
			Render(s.CurrentModel)
	}

	// Calculate spacing
	// We need the width to align right, but State doesn't have it easily accessible here
	// For now, just append with a space if model exists
	if rightSide != "" {
		return fmt.Sprintf("%s  %s", leftSide, rightSide)
	}
	return leftSide
}
