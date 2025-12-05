package views

import (
	"testing"

	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/stretchr/testify/assert"
)

func TestRenderModelPopup_WithSelection(t *testing.T) {
	state := models.State{
		ShowModelList:  true,
		ModelList:      []string{"gemini-2.5-pro", "gemini-2.5-flash"},
		ModelListIndex: 1,
	}

	result := RenderModelPopup(state)

	assert.Contains(t, result, "Select Model")
	assert.Contains(t, result, "gemini-2.5-pro")
	assert.Contains(t, result, "▸ gemini-2.5-flash")
	assert.Contains(t, result, "Navigate")
}

func TestRenderModelPopup_EmptyList(t *testing.T) {
	state := models.State{
		ShowModelList: true,
		ModelList:     []string{},
	}

	result := RenderModelPopup(state)

	assert.Empty(t, result)
}

func TestRenderModelPopup_IndexOutOfBounds(t *testing.T) {
	state := models.State{
		ShowModelList:  true,
		ModelList:      []string{"a", "b"},
		ModelListIndex: 10,
	}

	result := RenderModelPopup(state)

	assert.Contains(t, result, "a")
	assert.Contains(t, result, "b")
	assert.NotContains(t, result, "▸") // No highlight
}
