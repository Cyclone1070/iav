package ui

import (
	"context"

	"github.com/Cyclone1070/deployforme/internal/ui/models"
	"github.com/Cyclone1070/deployforme/internal/ui/services"
	tea "github.com/charmbracelet/bubbletea"
)

// UI implements the UserInterface using Bubble Tea
type UI struct {
	program *tea.Program

	// Orchestrator -> UI channels
	inputReq      chan inputRequest
	inputResp     chan string
	permReq       chan permRequest
	permResp      chan PermissionDecision
	statusChan    chan statusMsg
	messageChan   chan string
	modelListChan chan []string

	// UI -> Orchestrator
	commandChan chan UICommand

	// Ready signal
	readyChan chan struct{}
}

// Internal message types
type inputRequest struct {
	prompt string
}

type permRequest struct {
	prompt  string
	preview *models.ToolPreview
}

type statusMsg struct {
	phase   string
	message string
}

// UIChannels holds the channels for UI communication
type UIChannels struct {
	InputReq      chan inputRequest
	InputResp     chan string
	PermReq       chan permRequest
	PermResp      chan PermissionDecision
	StatusChan    chan statusMsg
	MessageChan   chan string
	ModelListChan chan []string
	CommandChan   chan UICommand
	ReadyChan     chan struct{} // Signals when UI is ready to accept requests
}

// NewUIChannels creates a new UIChannels struct with default buffers
func NewUIChannels() *UIChannels {
	return &UIChannels{
		InputReq:      make(chan inputRequest),
		InputResp:     make(chan string),
		PermReq:       make(chan permRequest),
		PermResp:      make(chan PermissionDecision),
		StatusChan:    make(chan statusMsg, 10),
		MessageChan:   make(chan string, 10),
		ModelListChan: make(chan []string),
		CommandChan:   make(chan UICommand, 10),
		ReadyChan:     make(chan struct{}),
	}
}

// NewUI creates a new Bubble Tea UI
func NewUI(
	channels *UIChannels,
	renderer services.MarkdownRenderer,
	spinnerFactory SpinnerFactory,
) *UI {
	ui := &UI{
		inputReq:      channels.InputReq,
		inputResp:     channels.InputResp,
		permReq:       channels.PermReq,
		permResp:      channels.PermResp,
		statusChan:    channels.StatusChan,
		messageChan:   channels.MessageChan,
		modelListChan: channels.ModelListChan,
		commandChan:   channels.CommandChan,
		readyChan:     channels.ReadyChan,
	}

	model := newBubbleTeaModel(
		ui.inputReq,
		ui.inputResp,
		ui.permReq,
		ui.permResp,
		ui.statusChan,
		ui.messageChan,
		ui.modelListChan,
		ui.commandChan,
		ui.readyChan,
		renderer,
		spinnerFactory,
	)

	ui.program = tea.NewProgram(model, tea.WithAltScreen())

	return ui
}

// Start starts the UI program
func (u *UI) Start() error {
	_, err := u.program.Run()
	return err
}

// ReadInput prompts the user for input
func (u *UI) ReadInput(ctx context.Context, prompt string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case u.inputReq <- inputRequest{prompt: prompt}:
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case response := <-u.inputResp:
			return response, nil
		}
	}
}

// ReadPermission prompts the user for a permission decision
func (u *UI) ReadPermission(ctx context.Context, prompt string, preview *models.ToolPreview) (PermissionDecision, error) {
	select {
	case <-ctx.Done():
		return DecisionDeny, ctx.Err()
	case u.permReq <- permRequest{prompt: prompt, preview: preview}:
		select {
		case <-ctx.Done():
			return DecisionDeny, ctx.Err()
		case decision := <-u.permResp:
			return decision, nil
		}
	}
}

// WriteStatus updates the status bar
func (u *UI) WriteStatus(phase string, message string) {
	select {
	case u.statusChan <- statusMsg{phase: phase, message: message}:
	default:
		// Drop if channel is full
	}
}

// WriteMessage sends a message to the UI
func (u *UI) WriteMessage(content string) {
	select {
	case u.messageChan <- content:
	default:
		// Drop if channel is full
	}
}

// WriteModelList sends a list of models to the UI
func (u *UI) WriteModelList(models []string) {
	select {
	case u.modelListChan <- models:
	default:
		// Drop if channel is full
	}
}

// Commands returns the command channel
func (u *UI) Commands() <-chan UICommand {
	return u.commandChan
}

// Ready returns a channel that is closed when the UI is ready to accept requests
func (u *UI) Ready() <-chan struct{} {
	return u.readyChan
}
