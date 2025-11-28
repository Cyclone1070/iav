package ui

import (
	"testing"
	"time"

	"github.com/Cyclone1070/deployforme/internal/ui/models"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

func createTestModel() BubbleTeaModel {
	channels := NewUIChannels()
	return newBubbleTeaModel(
		channels.InputReq,
		channels.InputResp,
		channels.PermReq,
		channels.PermResp,
		channels.StatusChan,
		channels.MessageChan,
		channels.ModelListChan,
		channels.CommandChan,
		channels.ReadyChan,
		&MockMarkdownRenderer{},
		mockSpinnerFactory,
	)
}

func TestInit_ReturnsCommands(t *testing.T) {
	model := createTestModel()
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestUpdate_KeyEnter_SubmitsInput(t *testing.T) {
	model := createTestModel()
	model.state.Input.SetValue("hello")
	model.state.CanSubmit = true

	// Capture response
	respChan := make(chan string, 1)
	model.inputResp = respChan

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
	m := newModel.(BubbleTeaModel)

	assert.Equal(t, "", m.state.Input.Value())
	assert.False(t, m.state.CanSubmit)
	assert.Len(t, m.state.Messages, 1)
	assert.Equal(t, "user", m.state.Messages[0].Role)
	assert.Equal(t, "hello", m.state.Messages[0].Content)

	select {
	case resp := <-respChan:
		assert.Equal(t, "hello", resp)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for response")
	}
}

func TestUpdate_SlashModels_OpensPopup(t *testing.T) {
	model := createTestModel()
	model.state.Input.SetValue("/models")
	model.state.CanSubmit = true

	cmdChan := make(chan UICommand, 1)
	model.commandChan = cmdChan

	msg := tea.KeyMsg{Type: tea.KeyEnter}
	newModel, _ := model.Update(msg)
	m := newModel.(BubbleTeaModel)

	assert.Equal(t, "", m.state.Input.Value())

	select {
	case cmd := <-cmdChan:
		assert.Equal(t, "list_models", cmd.Type)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for command")
	}
}

func TestUpdate_PopupNavigation_Down(t *testing.T) {
	model := createTestModel()
	model.state.ShowModelList = true
	model.state.ModelList = []string{"a", "b", "c"}
	model.state.ModelListIndex = 0

	msg := tea.KeyMsg{Type: tea.KeyDown}
	newModel, _ := model.Update(msg)
	m := newModel.(BubbleTeaModel)

	assert.Equal(t, 1, m.state.ModelListIndex)
}

func TestUpdate_PermissionYes(t *testing.T) {
	model := createTestModel()
	model.state.PendingPermission = &models.PermissionRequest{Prompt: "Allow?"}

	permChan := make(chan PermissionDecision, 1)
	model.permResp = permChan

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	newModel, _ := model.Update(msg)
	m := newModel.(BubbleTeaModel)

	assert.Nil(t, m.state.PendingPermission)

	select {
	case decision := <-permChan:
		assert.Equal(t, DecisionAllow, decision)
	case <-time.After(100 * time.Millisecond):
		t.Error("Timeout waiting for decision")
	}
}

func TestTick_DotAnimation(t *testing.T) {
	model := createTestModel()
	model.state.DotCount = 0

	// Tick 4 times
	for i := 0; i < 4; i++ {
		msg := tickMsg(time.Now())
		newModel, _ := model.Update(msg)
		model = newModel.(BubbleTeaModel)
	}

	assert.Equal(t, 0, model.state.DotCount) // Cycles back to 0
}
