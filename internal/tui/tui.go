package tui

import (
	"strings"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type conversationLine struct {
	content string
	style   lipgloss.Style
}

type agentEventMsg agent.AgentEvent

type Model struct {
	events    <-chan agent.AgentEvent
	submit    chan<- string
	messages  []conversationLine
	viewport  viewport.Model
	ready     bool
	width     int
	height    int
	input     string
	streaming bool
}

func New(events <-chan agent.AgentEvent, submit chan<- string) *Model {
	return &Model{
		events:   events,
		submit:   submit,
		messages: []conversationLine{},
	}
}

func (m *Model) Init() tea.Cmd {
	return m.waitForEvent
}

func (m *Model) waitForEvent() tea.Msg {
	ev, ok := <-m.events
	if !ok {
		return tea.Quit()
	}
	return agentEventMsg(ev)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if !m.ready {
			m.viewport = viewport.New(msg.Width, msg.Height-3)
			m.ready = true
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = msg.Height - 3
		}
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if strings.TrimSpace(m.input) != "" {
				m.appendUserMessage(strings.TrimSpace(m.input))
				m.submit <- strings.TrimSpace(m.input)
				m.input = ""
			}
		case "backspace":
			if len(m.input) > 0 {
				m.input = m.input[:len(m.input)-1]
			}
		default:
			if len(msg.Runes) > 0 {
				m.input += string(msg.Runes)
			}
		}

	case agentEventMsg:
		m.handleEvent(agent.AgentEvent(msg))
		return m, m.waitForEvent
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) handleEvent(ev agent.AgentEvent) {
	switch ev.Type {
	case agent.EventModelResponseChunk:
		m.appendChunk(ev.Content)
	case agent.EventToolCallStarted:
		m.streaming = false
		m.appendToolCallStarted(ev.ToolCall)
	case agent.EventToolCallFinished:
		m.appendToolCallFinished(ev.ToolCall, ev.ToolResult)
	case agent.EventError:
		m.streaming = false
		m.appendError(ev.Error)
	}
}

func (m *Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	inputPrompt := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#faa968")).
		Render("> ")

	inputLine := inputPrompt + m.input

	inputStyle := lipgloss.NewStyle().
		Width(m.width).
		Background(lipgloss.Color("#00172e")).
		Foreground(lipgloss.Color("#f6dcac"))

	return lipgloss.JoinVertical(
		lipgloss.Top,
		m.viewport.View(),
		inputStyle.Render(inputLine),
	)
}

func (m *Model) renderMessages() string {
	var b strings.Builder
	wrapWidth := m.viewport.Width
	if wrapWidth <= 0 {
		wrapWidth = 80
	}
	for i, l := range m.messages {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(l.style.MaxWidth(wrapWidth).Render(l.content))
	}
	return b.String()
}

func (m *Model) appendUserMessage(content string) {
	userLabelStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#faa968")).
		Bold(true)
	userContentStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f6dcac"))

	m.messages = append(m.messages, conversationLine{content: "You", style: userLabelStyle})
	m.messages = append(m.messages, conversationLine{content: content, style: userContentStyle})
	m.streaming = false
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m *Model) appendChunk(content string) {
	if m.streaming {
		m.messages[len(m.messages)-1].content += content
	} else {
		assistantLabelStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f6dcac")).
			Bold(true)
		m.messages = append(m.messages, conversationLine{content: "Assistant", style: assistantLabelStyle})
		assistantStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#f6dcac"))
		m.messages = append(m.messages, conversationLine{content: content, style: assistantStyle})
		m.streaming = true
	}
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m *Model) appendToolCallStarted(call *core.ToolCall) {
	if call == nil {
		return
	}
	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3f8f8a")).
		Italic(true)

	m.messages = append(m.messages, conversationLine{
		content: "  " + call.Name + "(" + call.Arguments + ")",
		style:   style,
	})
	m.messages = append(m.messages, conversationLine{
		content: "  … running",
		style:   lipgloss.NewStyle().Foreground(lipgloss.Color("#a7c9c6")),
	})
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m *Model) appendToolCallFinished(call *core.ToolCall, result *core.ToolResult) {
	if call == nil || result == nil {
		return
	}

	// Remove the "… running" line
	if len(m.messages) > 0 && m.messages[len(m.messages)-1].content == "  … running" {
		m.messages = m.messages[:len(m.messages)-1]
	}

	isError := result.IsError
	resultStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#8cbfb8"))

	status := "  → " + result.Content
	if isError {
		status = "  ✗ " + result.Content
		resultStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#f85525"))
	}

	m.messages = append(m.messages, conversationLine{
		content: status,
		style:   resultStyle,
	})
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}

func (m *Model) appendError(err error) {
	if err == nil {
		return
	}
	errStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#f85525")).
		Bold(true)

	m.messages = append(m.messages, conversationLine{
		content: "  ✗ Error: " + err.Error(),
		style:   errStyle,
	})
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()
}
