package tui

import (
	"strings"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type agentEventMsg agent.AgentEvent

type Model struct {
	events   <-chan agent.AgentEvent
	submit   chan<- string
	viewport viewport.Model
	spinner  spinner.Model
	help     help.Model
	ready    bool
	started  bool
	width    int
	height   int
	bubbles  []messageBubble
	status   statusData
	showHelp bool
	input    string

	streaming bool
	thinking  bool

	toolCards      []toolCallCard
	cardsExpanded  bool
}

func New(events <-chan agent.AgentEvent, submit chan<- string) *Model {
	s := spinner.New()
	s.Style = SpinnerStyle
	s.Spinner = spinner.Dot

	return &Model{
		events:   events,
		submit:   submit,
		spinner:  s,
		help:     newHelpModel(),
		status: statusData{
			modelName:    "",
			tokenCount:   0,
			sandboxState: "",
		},
		bubbles:       []messageBubble{},
		toolCards:     []toolCallCard{},
		showHelp:      false,
		thinking:      false,
		input:         "",
	}
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(m.waitForEvent, m.spinner.Tick)
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
		m.handleResize(msg)

	case tea.KeyMsg:
		if m.showHelp {
			switch msg.String() {
			case "ctrl+h", "ctrl+c", "enter", "esc":
				m.showHelp = false
			}
			return m, nil
		}

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Help):
			m.showHelp = !m.showHelp
			return m, nil
		case key.Matches(msg, keys.Send):
			return m, m.sendMessage()
		case key.Matches(msg, keys.Collapse):
			m.cardsExpanded = !m.cardsExpanded
			for i := range m.toolCards {
				m.toolCards[i].collapsed = !m.cardsExpanded
			}
			m.updateViewport()
		default:
			m.handleInput(msg)
		}

	case agentEventMsg:
		m.handleEvent(agent.AgentEvent(msg))
		return m, m.waitForEvent

	case spinner.TickMsg:
		if m.thinking {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			m.updateViewport()
			return m, cmd
		}
	}

	var cmds []tea.Cmd
	var cmd tea.Cmd

	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	if m.thinking {
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) handleInput(msg tea.KeyMsg) {
	switch msg.String() {
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			m.input += string(msg.Runes)
		}
	}
}

func (m *Model) handleResize(msg tea.WindowSizeMsg) {
	headerHeight := 1
	statusHeight := 2
	inputHeight := 2
	helpHeight := 0
	if m.showHelp {
		helpHeight = 8
	}

	availableHeight := msg.Height - headerHeight - statusHeight - inputHeight - helpHeight
	if availableHeight < 10 {
		availableHeight = 10
	}

	if !m.ready {
		m.viewport = viewport.New(msg.Width, availableHeight)
		m.ready = true
	} else {
		m.viewport.Width = msg.Width
		m.viewport.Height = availableHeight
	}

	m.help.Width = msg.Width

	m.updateViewport()
}

func (m *Model) sendMessage() tea.Cmd {
	text := strings.TrimSpace(m.input)
	if text == "" {
		return nil
	}

	if !m.started {
		m.started = true
		m.viewport.GotoBottom()
	}

	m.appendUserMessage(text)
	m.submit <- text
	m.input = ""
	return nil
}

func (m *Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	if !m.started {
		return m.renderWelcome()
	}

	header := m.renderHeader()
	viewportView := ViewportStyle.Render(m.viewport.View())
	statusBar := m.renderStatusBar()
	inputView := InputBarStyle.Render(renderInput("> ", m.input, "█", m.width))

	parts := []string{header, viewportView, statusBar, inputView}

	if m.showHelp {
		helpView := m.help.View(keys)
		parts = append(parts, helpView)
	}

	return lipgloss.JoinVertical(lipgloss.Top, parts...)
}

func (m *Model) renderHeader() string {
	title := HeaderStyle.Render("LKG")
	modelInfo := ""
	if m.status.modelName != "" {
		modelInfo = HeaderBarStyle.Render(" \u00b7 " + m.status.modelName)
	}
	spacer := HeaderBarStyle.Width(m.width - lipgloss.Width(title) - lipgloss.Width(modelInfo) - 2).Render(strings.Repeat(" ", m.width))
	return lipgloss.JoinHorizontal(lipgloss.Top, title, modelInfo, spacer)
}

func (m *Model) renderWelcome() string {
	topPad := m.height / 3
	if topPad < 2 {
		topPad = 2
	}

	spacing := strings.Repeat("\n", topPad-1)

	title := WelcomeTitleStyle.Render("LKG")
	subtitle := WelcomeSubtitleStyle.Render("Last Known Good \u2014 the open source AI coding agent")
	inputLine := renderInput("> ", m.input, "█", 60)

	return lipgloss.JoinVertical(
		lipgloss.Center,
		spacing,
		"",
		title,
		"",
		subtitle,
		"",
		"",
		inputLine,
	)
}

func (m *Model) appendUserMessage(content string) {
	m.bubbles = append(m.bubbles, messageBubble{
		msgType: msgUser,
		content: content,
		label:   "You",
		model:   m.status.modelName,
	})
	m.thinking = true
	m.updateViewport()
	m.viewport.GotoBottom()
}

func (m *Model) handleEvent(ev agent.AgentEvent) {
	switch ev.Type {
	case agent.EventModelResponseChunk:
		m.handleChunk(ev.Content)
	case agent.EventToolCallStarted:
		m.handleToolCallStarted(ev.ToolCall)
	case agent.EventToolCallFinished:
		m.handleToolCallFinished(ev.ToolCall, ev.ToolResult)
	case agent.EventTurnComplete:
		m.thinking = false
		m.streaming = false
		m.updateViewport()
	case agent.EventError:
		m.handleError(ev.Error)
	}
}

func (m *Model) handleChunk(content string) {
	if !m.started {
		m.started = true
	}
	if !m.streaming {
		m.thinking = false
		m.streaming = true
		m.bubbles = append(m.bubbles, messageBubble{
			msgType: msgAssistant,
			content: content,
			label:   "Assistant",
			model:   m.status.modelName,
		})
	} else if len(m.bubbles) > 0 {
		last := &m.bubbles[len(m.bubbles)-1]
		if last.msgType == msgAssistant {
			last.content += content
		}
	}
	m.status.tokenCount += len(content) / 4
	m.updateViewport()
	m.viewport.GotoBottom()
}

func (m *Model) handleToolCallStarted(call *core.ToolCall) {
	if call == nil {
		return
	}
	m.thinking = false
	m.streaming = false

	card := toolCallCard{
		name:      call.Name,
		arguments: call.Arguments,
		collapsed: !m.cardsExpanded,
	}
	m.toolCards = append(m.toolCards, card)

	m.bubbles = append(m.bubbles, messageBubble{
		msgType: msgToolCall,
		content: call.Name + "(" + call.Arguments + ")",
	})
	m.bubbles = append(m.bubbles, messageBubble{
		msgType: msgToolResult,
		content: "\u2026 running",
	})
	m.updateViewport()
	m.viewport.GotoBottom()
}

func (m *Model) handleToolCallFinished(call *core.ToolCall, result *core.ToolResult) {
	if call == nil || result == nil {
		return
	}

	for i := len(m.bubbles) - 1; i >= 0; i-- {
		if m.bubbles[i].msgType == msgToolResult && m.bubbles[i].content == "\u2026 running" {
			m.bubbles = append(m.bubbles[:i], m.bubbles[i+1:]...)
			break
		}
	}

	for i := range m.toolCards {
		if m.toolCards[i].name == call.Name {
			m.toolCards[i].result = result.Content
			m.toolCards[i].isError = result.IsError
			break
		}
	}

	status := "\u2192 " + result.Content
	if result.IsError {
		status = "\u2717 " + result.Content
	}

	m.bubbles = append(m.bubbles, messageBubble{
		msgType: msgToolResult,
		content: status,
	})
	m.updateViewport()
	m.viewport.GotoBottom()
}

func (m *Model) handleError(err error) {
	if err == nil {
		return
	}
	m.thinking = false
	m.streaming = false

	m.bubbles = append(m.bubbles, messageBubble{
		msgType: msgError,
		content: err.Error(),
	})
	m.updateViewport()
	m.viewport.GotoBottom()
}

func (m *Model) updateViewport() {
	var b strings.Builder
	for i, bubble := range m.bubbles {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(m.renderMessageBubble(bubble))
	}

	if m.thinking {
		if len(m.bubbles) > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(ThinkingStyle.Render(m.spinner.View() + " thinking..."))
	}

	m.viewport.SetContent(b.String())
}

func (m *Model) SetModelName(name string) {
	m.status.modelName = name
}

func (m *Model) SetSandboxState(state string) {
	m.status.sandboxState = state
}

func (m *Model) SetTokenCount(count int) {
	m.status.tokenCount = count
}
