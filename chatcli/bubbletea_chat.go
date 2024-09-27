package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	promptStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500"))
	inputStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	messageStyle = lipgloss.NewStyle().Margin(1, 2)
)

type chatModel struct {
	history     []Message
	input       textarea.Model
	chat        *Chat
	err         error
	isQuitting  bool
	viewport    viewport.Model
	width       int
	height      int
	currentMode tea.Model
}

func (m *chatModel) Init() tea.Cmd {
	m.input = textarea.New()
	m.input.Placeholder = "Type your message..."
	m.input.Focus()
	m.input.CharLimit = 500
	m.input.SetWidth(50)
	m.input.SetHeight(3)
	m.input.ShowLineNumbers = false

	m.viewport = viewport.New(0, 0) // Width and height will be set on window resize

	return tea.Batch(textarea.Blink, tea.EnterAltScreen)
}

// func modeCompleted(mode tea.Model) bool {
// 	// Implement logic to determine if the mode is completed
// 	return false // Placeholder
// }

func (m *chatModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// If a mode is active, delegate update to the mode's model
	if m.currentMode != nil {
		var cmd tea.Cmd
		m.currentMode, cmd = m.currentMode.Update(msg)
		// // Check if mode is completed
		// if modeCompleted(m.currentMode) {
		// 	m.currentMode = nil
		// }
		return m, cmd
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Adjust viewport size
		inputHeight := m.input.Height() + 2 // Extra space for borders/margins
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - inputHeight - 1 // Subtract input and a separator line

		// Update viewport content
		m.viewport.SetContent(m.renderHistory())
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.isQuitting = true
			return m, tea.Quit

		case tea.KeyEnter:
			userInput := m.input.Value()
			m.input.Reset()

			var response string
			var command Command
			var err error

			if m.chat.modeManager.currentMode != nil {
				response, command, err = m.chat.modeManager.HandleInput(userInput)
			} else {
				response, command, err = m.chat.HandleUserMessage(userInput)
			}

			if err != nil {
				m.err = err
			} else {
				m.history = append(m.history, Message{Sender: SenderYou, Content: userInput})
				m.history = append(m.history, Message{Sender: SenderAI, Content: response})
				m.err = nil

				// Update viewport content
				m.viewport.SetContent(m.renderHistory())
				m.viewport.GotoBottom()
			}

			if command == QUIT {
				m.isQuitting = true
				return m, tea.Quit
			}
		case tea.KeyUp:
			m.viewport.LineUp(1)

		case tea.KeyDown:
			m.viewport.LineDown(1)
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *chatModel) renderHistory() string {
	var historyBuilder strings.Builder

	// Get the maximum width for messages
	maxWidth := m.viewport.Width - 10 // Adjust based on desired margins

	// Define styles
	userStyle := lipgloss.NewStyle().
		Width(maxWidth).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#007ACC")). // A pleasant blue color
		Padding(1, 2).
		Margin(1, 2).
		Border(lipgloss.RoundedBorder()).
		Align(lipgloss.Right)

	aiStyle := lipgloss.NewStyle().
		Width(maxWidth).
		Foreground(lipgloss.Color("#FFFFFF")).
		/* A pleasant purple color*/ Background(lipgloss.Color("#800080")).
		Padding(1, 2).
		Margin(1, 2).
		Border(lipgloss.RoundedBorder()).
		Align(lipgloss.Left)

	// Build chat history
	for _, msg := range m.history {
		var styledMsg string
		messageContent := fmt.Sprintf("%s: %s", msg.Sender, msg.Content)

		if msg.Sender == SenderYou {
			styledMsg = userStyle.Render(messageContent)
		} else {
			styledMsg = aiStyle.Render(messageContent)
		}
		historyBuilder.WriteString(styledMsg + "\n")
	}

	return historyBuilder.String()
}

func (m *chatModel) View2() string {
	var sb strings.Builder

	// Define styles
	userStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#00FF00"))
	aiStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF69B4"))
	inputStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFFFF"))

	// Display chat history
	for _, msg := range m.history {
		var styledMsg string
		if msg.Sender == SenderYou {
			styledMsg = userStyle.Render(fmt.Sprintf("You: %s", msg.Content))
		} else {
			styledMsg = aiStyle.Render(fmt.Sprintf("AI: %s", msg.Content))
		}
		sb.WriteString(styledMsg + "\n")
	}

	// Display input prompt
	sb.WriteString("\n" + inputStyle.Render(m.input.View()))

	if m.err != nil {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
		sb.WriteString("\n" + errorStyle.Render("Error: "+m.err.Error()))
	}

	if m.isQuitting {
		sb.WriteString("\nGoodbye!\n")
	}

	return sb.String()
}
func (m *chatModel) View() string {
	if m.isQuitting {
		return "Goodbye!\n"
	}

	// Render input area
	inputView := m.input.View()

	// Combine viewport and input
	return fmt.Sprintf(
		"%s\n%s\n%s",
		m.viewport.View(),
		strings.Repeat("─", m.width),
		inputView,
	)
}

func mainBubbleTea(chat *Chat, shutdownChan chan struct{}) {
	// Create initial model
	m := &chatModel{
		chat: chat,
	}
	m.Init()

	// Start Bubble Tea program
	p := tea.NewProgram(m)

	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
