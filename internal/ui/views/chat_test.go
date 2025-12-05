package views

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderChat_NoMessages(t *testing.T) {
	state := models.State{Messages: []models.Message{}}
	result := RenderChat(state, &MockMarkdownRenderer{})
	assert.Contains(t, result, "No messages yet")
}

func TestRenderChat_WithMessages(t *testing.T) {
	// Since RenderChat just delegates to Viewport.View(), we test that it returns the viewport content
	vp := createTestViewport()
	vp.SetContent("Rendered Content")

	state := models.State{
		Messages: []models.Message{{Role: "user", Content: "Hello"}},
		Viewport: vp,
	}

	result := RenderChat(state, &MockMarkdownRenderer{})
	assert.Contains(t, result, "Rendered Content")
}
