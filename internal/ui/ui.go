package ui

import (
	"context"

	"github.com/Cyclone1070/iav/internal/config"
	"github.com/Cyclone1070/iav/internal/ui/models"
	"github.com/Cyclone1070/iav/internal/ui/services"
	tea "github.com/charmbracelet/bubbletea"
)

// UI implements the UserInterface using Bubble Tea
type UI struct {
	program *tea.Program

	// Orchestrator -> UI channels
	inputReq      chan InputRequest
	inputResp     chan string
	permReq       chan PermRequest
	permResp      chan PermissionDecision
	statusChan    chan StatusMsg
	messageChan   chan string
	modelListChan chan []string
	setModelChan  chan string

	// UI -> Orchestrator
	commandChan chan UICommand

	// Ready signal
	readyChan chan struct{}
}

// Internal message types
// Internal message types
type InputRequest struct {
	Prompt string
}

type PermRequest struct {
	Prompt  string
	Preview *models.ToolPreview
}

type StatusMsg struct {
	Phase   string
	Message string
}

// UIChannels holds the channels for UI communication
type UIChannels struct {
	InputReq      chan InputRequest
	InputResp     chan string
	PermReq       chan PermRequest
	PermResp      chan PermissionDecision
	StatusChan    chan StatusMsg
	MessageChan   chan string
	ModelListChan chan []string
	SetModelChan  chan string
	CommandChan   chan UICommand
	ReadyChan     chan struct{} // Signals when UI is ready to accept requests
}

// NewUIChannels creates a new UIChannels struct with buffer sizes from config
func NewUIChannels(cfg *config.Config) *UIChannels {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	return &UIChannels{
		InputReq:      make(chan InputRequest),
		InputResp:     make(chan string),
		PermReq:       make(chan PermRequest),
		PermResp:      make(chan PermissionDecision),
		StatusChan:    make(chan StatusMsg, cfg.UI.StatusChannelBuffer),
		MessageChan:   make(chan string, cfg.UI.MessageChannelBuffer),
		ModelListChan: make(chan []string),
		SetModelChan:  make(chan string, cfg.UI.SetModelChannelBuffer),
		CommandChan:   make(chan UICommand, cfg.UI.CommandChannelBuffer),
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
		setModelChan:  channels.SetModelChan,
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
		ui.setModelChan,
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
	case u.inputReq <- InputRequest{Prompt: prompt}:
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
	case u.permReq <- PermRequest{Prompt: prompt, Preview: preview}:
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
	case u.statusChan <- StatusMsg{Phase: phase, Message: message}:
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

// SetModel updates the current model name displayed in the UI
func (u *UI) SetModel(model string) {
	select {
	case u.setModelChan <- model:
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
