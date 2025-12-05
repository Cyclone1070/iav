package views

import (
	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/Cyclone1070/iav/internal/ui/services"
	"github.com/charmbracelet/lipgloss"
)

// RenderRoot renders the complete UI layout
func RenderRoot(s models.State, renderer services.MarkdownRenderer) string {
	sections := []string{
		RenderChat(s, renderer),
		RenderInput(s),
		RenderStatus(s),
	}

	// Add model popup if visible
	if s.ShowModelList {
		popup := RenderModelPopup(s)
		// Overlay popup on top
		return lipgloss.Place(
			s.Width,
			s.Height,
			lipgloss.Center,
			lipgloss.Center,
			popup,
			lipgloss.WithWhitespaceChars(""),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}
