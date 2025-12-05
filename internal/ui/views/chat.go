package views

import (
	"strings"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/Cyclone1070/iav/internal/ui/services"
)

// RenderChat renders the message history
func RenderChat(s models.State, renderer services.MarkdownRenderer) string {
	if len(s.Messages) == 0 {
		return "No messages yet. Type a message to start."
	}
	return s.Viewport.View()
}

// FormatChatContent formats the messages for the viewport
func FormatChatContent(messages []models.Message, width int, renderer services.MarkdownRenderer) string {
	var lines []string
	for _, msg := range messages {
		if msg.Role == "user" {
			lines = append(lines, UserMessageStyle.Render("You: "+msg.Content))
		} else {
			// Render assistant messages as markdown
			rendered, err := services.RenderMarkdown(msg.Content, width, renderer)
			if err != nil {
				// Fallback to plain text
				lines = append(lines, AssistantMessageStyle.Render("AI: "+msg.Content))
			} else {
				lines = append(lines, AssistantMessageStyle.Render(rendered))
			}
		}
		lines = append(lines, "") // Add spacing
	}
	return strings.Join(lines, "\n")
}
