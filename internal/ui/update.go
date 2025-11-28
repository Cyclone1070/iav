package ui

import (
	"strings"
	"time"

	"github.com/Cyclone1070/deployforme/internal/ui/models"
	"github.com/Cyclone1070/deployforme/internal/ui/services"
	"github.com/Cyclone1070/deployforme/internal/ui/views"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// BubbleTeaModel implements tea.Model
type BubbleTeaModel struct {
	state models.State

	// Dependencies
	renderer services.MarkdownRenderer

	// Channels for communication with orchestrator
	inputReq      <-chan inputRequest
	inputResp     chan<- string
	permReq       <-chan permRequest
	permResp      chan<- PermissionDecision
	statusChan    <-chan statusMsg
	messageChan   <-chan string
	modelListChan <-chan []string

	// UI -> Orchestrator
	commandChan chan<- UICommand

	// Ready signal
	readyChan chan<- struct{}
}

// ... (Init and Update methods remain mostly same, just View changes)

// View renders the UI
func (m BubbleTeaModel) View() string {
	return views.RenderRoot(m.state, m.renderer)
}

// ... (helpers)

// SpinnerFactory creates a new spinner
type SpinnerFactory func() spinner.Model

// newBubbleTeaModel creates a new Bubble Tea model
func newBubbleTeaModel(
	inputReq <-chan inputRequest,
	inputResp chan<- string,
	permReq <-chan permRequest,
	permResp chan<- PermissionDecision,
	statusChan <-chan statusMsg,
	messageChan <-chan string,
	modelListChan <-chan []string,
	commandChan chan<- UICommand,
	readyChan chan<- struct{},
	renderer services.MarkdownRenderer,
	spinnerFactory SpinnerFactory,
) BubbleTeaModel {
	// Initialize components
	ti := textinput.New()
	ti.Placeholder = "Type a message..."
	ti.Focus()

	vp := viewport.New(80, 20)

	sp := spinnerFactory()

	return BubbleTeaModel{
		state: models.State{
			Input:    ti,
			Viewport: vp,
			Spinner:  sp,
			Messages: []models.Message{},
		},
		renderer:      renderer,
		inputReq:      inputReq,
		inputResp:     inputResp,
		permReq:       permReq,
		permResp:      permResp,
		statusChan:    statusChan,
		messageChan:   messageChan,
		modelListChan: modelListChan,
		commandChan:   commandChan,
		readyChan:     readyChan,
	}
}

// Internal messages
type tickMsg time.Time
type inputRequestMsg inputRequest
type permRequestMsg permRequest
type statusUpdateMsg statusMsg
type messageReceivedMsg string
type modelListReceivedMsg []string

// Init initializes the model
func (m BubbleTeaModel) Init() tea.Cmd {
	// Signal that UI is ready
	if m.readyChan != nil {
		close(m.readyChan)
	}

	return tea.Batch(
		textinput.Blink,
		m.state.Spinner.Tick,
		tick(),
		listenForInputRequests(m.inputReq),
		listenForPermRequests(m.permReq),
		listenForStatus(m.statusChan),
		listenForMessages(m.messageChan),
		listenForModelList(m.modelListChan),
	)
}

// Update handles messages
func (m BubbleTeaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.state.Width = msg.Width
		m.state.Height = msg.Height
		m.state.Viewport.Width = msg.Width
		m.state.Viewport.Height = msg.Height - 6 // Reserve space for input and status

	case tickMsg:
		// Update dot animation
		m.state.DotCount = (m.state.DotCount + 1) % 4
		var cmd tea.Cmd
		m.state.Spinner, cmd = m.state.Spinner.Update(msg)
		return m, tea.Batch(cmd, tick())

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.state.Spinner, cmd = m.state.Spinner.Update(msg)
		return m, cmd

	case inputRequestMsg:
		m.state.CanSubmit = true
		return m, listenForInputRequests(m.inputReq)

	case permRequestMsg:
		m.state.PendingPermission = &models.PermissionRequest{
			Prompt:  msg.prompt,
			Preview: msg.preview,
		}
		return m, listenForPermRequests(m.permReq)

	case statusUpdateMsg:
		m.state.StatusPhase = msg.phase
		m.state.StatusMessage = msg.message
		return m, listenForStatus(m.statusChan)

	case messageReceivedMsg:
		m.state.Messages = append(m.state.Messages, models.Message{
			Role:    "assistant",
			Content: string(msg),
		})
		m.updateViewport()
		return m, listenForMessages(m.messageChan)

	case modelListReceivedMsg:
		m.state.ModelList = []string(msg)
		m.state.ShowModelList = true
		m.state.ModelListIndex = 0
		return m, listenForModelList(m.modelListChan)
	}

	// Update input
	var cmd tea.Cmd
	m.state.Input, cmd = m.state.Input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// handleKeyPress handles keyboard input
func (m BubbleTeaModel) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle model popup navigation
	if m.state.ShowModelList {
		switch msg.String() {
		case "up", "k":
			if m.state.ModelListIndex > 0 {
				m.state.ModelListIndex--
			}
		case "down", "j":
			if m.state.ModelListIndex < len(m.state.ModelList)-1 {
				m.state.ModelListIndex++
			}
		case "enter":
			// Send switch model command
			if m.state.ModelListIndex < len(m.state.ModelList) {
				m.commandChan <- UICommand{
					Type: "switch_model",
					Args: map[string]string{
						"model": m.state.ModelList[m.state.ModelListIndex],
					},
				}
			}
			m.state.ShowModelList = false
		case "esc":
			m.state.ShowModelList = false
		}
		return m, nil
	}

	// Handle permission prompts
	if m.state.PendingPermission != nil {
		switch msg.String() {
		case "y":
			m.permResp <- DecisionAllow
			m.state.PendingPermission = nil
		case "n":
			m.permResp <- DecisionDeny
			m.state.PendingPermission = nil
		case "a":
			m.permResp <- DecisionAllowAlways
			m.state.PendingPermission = nil
		}
		return m, nil
	}

	// Handle normal input
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit

	case "enter":
		if m.state.CanSubmit && m.state.Input.Value() != "" {
			input := m.state.Input.Value()

			// Check for commands
			if strings.HasPrefix(input, "/") {
				return m.handleCommand(input)
			}

			// Send user message
			m.state.Messages = append(m.state.Messages, models.Message{
				Role:    "user",
				Content: input,
			})
			m.updateViewport()

			// Send to orchestrator
			m.inputResp <- input
			m.state.Input.SetValue("")
			m.state.CanSubmit = false
		}
	}

	return m, nil
}

// handleCommand handles slash commands
func (m BubbleTeaModel) handleCommand(input string) (tea.Model, tea.Cmd) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return m, nil
	}

	cmd := parts[0]
	switch cmd {
	case "/models":
		// Request model list
		m.commandChan <- UICommand{Type: "list_models"}
		m.state.Input.SetValue("")
	case "/help":
		m.state.Messages = append(m.state.Messages, models.Message{
			Role:    "assistant",
			Content: "Available commands:\n- /models - List and switch models\n- /help - Show this help",
		})
		m.updateViewport()
		m.state.Input.SetValue("")
	}

	return m, nil
}

// updateViewport updates the viewport content
func (m *BubbleTeaModel) updateViewport() {
	content := views.FormatChatContent(m.state.Messages, m.state.Width-4, m.renderer)
	m.state.Viewport.SetContent(content)
	m.state.Viewport.GotoBottom()
}

// Helper commands for listening to channels
func listenForInputRequests(ch <-chan inputRequest) tea.Cmd {
	return func() tea.Msg {
		return inputRequestMsg(<-ch)
	}
}

func listenForPermRequests(ch <-chan permRequest) tea.Cmd {
	return func() tea.Msg {
		return permRequestMsg(<-ch)
	}
}

func listenForStatus(ch <-chan statusMsg) tea.Cmd {
	return func() tea.Msg {
		return statusUpdateMsg(<-ch)
	}
}

func listenForMessages(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		return messageReceivedMsg(<-ch)
	}
}

func listenForModelList(ch <-chan []string) tea.Cmd {
	return func() tea.Msg {
		return modelListReceivedMsg(<-ch)
	}
}

func tick() tea.Cmd {
	return tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}
