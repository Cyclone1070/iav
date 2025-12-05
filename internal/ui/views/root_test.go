package views

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderRoot_NormalState(t *testing.T) {
	messages := []models.Message{{Role: "user", Content: "Hi"}}
	renderer := &MockMarkdownRenderer{}

	vp := createTestViewport()
	vp.SetContent(FormatChatContent(messages, 76, renderer))

	state := models.State{
		Width:       80,
		Height:      24,
		Messages:    messages,
		Input:       createTestTextInput("typing..."),
		StatusPhase: "ready",
		Viewport:    vp,
	}

	result := RenderRoot(state, renderer)

	assert.Contains(t, result, "Hi")
	assert.Contains(t, result, "typing...")
	assert.Contains(t, result, "Ready")
}

func TestRenderRoot_WithPopup(t *testing.T) {
	state := models.State{
		Width:         80,
		Height:        24,
		ShowModelList: true,
		ModelList:     []string{"a", "b"},
		Input:         createTestTextInput(""),
		Viewport:      createTestViewport(),
	}

	result := RenderRoot(state, &MockMarkdownRenderer{})

	assert.Contains(t, result, "Select Model")
	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
}
