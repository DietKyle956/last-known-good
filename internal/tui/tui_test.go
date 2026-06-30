package tui

import (
	"errors"
	"testing"
	"time"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	tea "github.com/charmbracelet/bubbletea"
)

func TestModelImplementsTeaModel(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	var _ tea.Model = m
}

func TestModelInitialState(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	if m == nil {
		t.Fatal("New returned nil")
	}
}

func TestModelRendersHeader(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	v := m.View()
	if v == "" {
		t.Fatal("View returned empty string")
	}
	if len(v) < 10 {
		t.Fatalf("View too short: %q", v)
	}
}

func TestAppendChunkCreatesMessage(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.handleEvent(agent.AgentEvent{
		Type:    agent.EventModelResponseChunk,
		Content: "Hello",
	})

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].msgType != msgAssistant {
		t.Fatalf("expected assistant message type, got %d", m.bubbles[0].msgType)
	}
	if m.bubbles[0].content != "Hello" {
		t.Fatalf("expected content 'Hello', got %q", m.bubbles[0].content)
	}
}

func TestAppendMultipleChunksSameMessage(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.handleEvent(agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "Hel"})
	m.handleEvent(agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "lo"})
	m.handleEvent(agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: " world"})

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].content != "Hello world" {
		t.Fatalf("expected content 'Hello world', got %q", m.bubbles[0].content)
	}
}

func TestEventsChannelConsumedByModel(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "Streaming "}
	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "text"}

	msg1 := m.waitForEvent()
	m.Update(msg1)
	msg2 := m.waitForEvent()
	m.Update(msg2)

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].content != "Streaming text" {
		t.Fatalf("expected 'Streaming text', got %q", m.bubbles[0].content)
	}
}

func TestSubmitChannelSendsPrompt(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.input = "hello"
	m.sendMessage()

	select {
	case prompt := <-submit:
		if prompt != "hello" {
			t.Fatalf("expected 'hello', got %q", prompt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt on submit channel")
	}

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].msgType != msgUser {
		t.Fatalf("expected user message type, got %d", m.bubbles[0].msgType)
	}
	if m.bubbles[0].content != "hello" {
		t.Fatalf("expected content 'hello', got %q", m.bubbles[0].content)
	}
}

func TestToolCallStartedAddsLine(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallStarted,
		ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`},
	})

	if len(m.bubbles) < 2 {
		t.Fatalf("expected at least 2 bubbles, got %d", len(m.bubbles))
	}
	if m.bubbles[0].msgType != msgToolCall {
		t.Fatalf("expected tool call message type, got %d", m.bubbles[0].msgType)
	}
	if !contains(m.bubbles[0].content, "read_file") {
		t.Fatalf("expected tool name in first bubble, got %q", m.bubbles[0].content)
	}
	if m.bubbles[1].content != "… running" {
		t.Fatalf("expected running indicator, got %q", m.bubbles[1].content)
	}
}

func TestToolCallFinishedUpdatesLine(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallStarted,
		ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`},
	})

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallFinished,
		ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`},
		ToolResult: &core.ToolResult{
			ToolCallID: "c1",
			Content:    "file contents",
			IsError:    false,
		},
	})

	lastBubble := m.bubbles[len(m.bubbles)-1]
	if lastBubble.msgType != msgToolResult {
		t.Fatalf("expected tool result message type, got %d", lastBubble.msgType)
	}
	if !contains(lastBubble.content, "file contents") {
		t.Fatalf("expected result in last bubble, got %q", lastBubble.content)
	}
}

func TestToolCallErrorShowsExpanded(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallStarted,
		ToolCall: &core.ToolCall{ID: "c1", Name: "bash", Arguments: `{"cmd":"rm -rf /"}`},
	})

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallFinished,
		ToolCall: &core.ToolCall{ID: "c1", Name: "bash", Arguments: `{"cmd":"rm -rf /"}`},
		ToolResult: &core.ToolResult{
			ToolCallID: "c1",
			Content:    "permission denied",
			IsError:    true,
		},
	})

	lastBubble := m.bubbles[len(m.bubbles)-1]
	if !contains(lastBubble.content, "permission denied") {
		t.Fatalf("expected error message in last bubble, got %q", lastBubble.content)
	}
}

func TestErrorEventAddsErrorMessage(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:  agent.EventError,
		Error: errors.New("something went wrong"),
	})

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 error bubble, got %d", len(m.bubbles))
	}
	if !contains(m.bubbles[0].content, "something went wrong") {
		t.Fatalf("expected error in bubble, got %q", m.bubbles[0].content)
	}
}

func TestTurnCompleteStopsThinking(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.thinking = true
	m.handleEvent(agent.AgentEvent{
		Type: agent.EventTurnComplete,
	})

	if m.thinking {
		t.Fatal("expected thinking to be false after turn complete")
	}
}

func TestSetModelName(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.SetModelName("deepseek-v4-flash")

	if m.status.modelName != "deepseek-v4-flash" {
		t.Fatalf("expected model name 'deepseek-v4-flash', got %q", m.status.modelName)
	}
}

func TestSetSandboxState(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.SetSandboxState("running")

	if m.status.sandboxState != "running" {
		t.Fatalf("expected sandbox state 'running', got %q", m.status.sandboxState)
	}
}

func TestSetTokenCount(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.SetTokenCount(1234)

	if m.status.tokenCount != 1234 {
		t.Fatalf("expected token count 1234, got %d", m.status.tokenCount)
	}
}

func TestStatusBarRendersWithData(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.SetModelName("deepseek-v4-flash")
	m.SetSandboxState("running")
	m.SetTokenCount(1234)

	status := m.renderStatusBar()
	if !contains(status, "deepseek-v4-flash") {
		t.Fatalf("expected model name in status bar, got %q", status)
	}
	if !contains(status, "1234") {
		t.Fatalf("expected token count in status bar, got %q", status)
	}
	if !contains(status, "running") {
		t.Fatalf("expected sandbox state in status bar, got %q", status)
	}
}

func TestUserBubbleRenders(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.appendUserMessage("hello world")

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].msgType != msgUser {
		t.Fatalf("expected user message type")
	}

	rendered := m.renderMessageBubble(m.bubbles[0])
	if !contains(rendered, "hello world") {
		t.Fatalf("expected content in rendered bubble, got %q", rendered)
	}
}

func TestAssistantBubbleRenders(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.status.modelName = "deepseek-v4-flash"
	m.handleEvent(agent.AgentEvent{
		Type:    agent.EventModelResponseChunk,
		Content: "Hello world",
	})

	if len(m.bubbles) != 1 {
		t.Fatalf("expected 1 bubble, got %d", len(m.bubbles))
	}
	if m.bubbles[0].msgType != msgAssistant {
		t.Fatalf("expected assistant message type")
	}

	rendered := m.renderMessageBubble(m.bubbles[0])
	if !contains(rendered, "Hello world") {
		t.Fatalf("expected content in rendered bubble, got %q", rendered)
	}
}

func TestMarkdownCodeBlockHighlighting(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	text := "Here is some code:\n```go\npackage main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n```\nEnd."

	rendered := m.renderMarkdown(text)
	if rendered == "" {
		t.Fatal("rendered markdown should not be empty")
	}
	// Verify text content appears after ANSI rendering
	if !contains(rendered, "Here is some code") {
		t.Fatalf("expected pre-code text, got %q", rendered)
	}
	if !contains(rendered, "End.") {
		t.Fatalf("expected post-code text, got %q", rendered)
	}
}

func TestTextWrapsAtViewportWidth(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:    agent.EventModelResponseChunk,
		Content: "This is a very long piece of text that should definitely be wrapped at the viewport width",
	})

	m.updateViewport()
}

func TestWelcomeScreenRenders(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	if m.started {
		t.Fatal("expected started to be false initially")
	}

	v := m.View()
	if !contains(v, "Last Known Good") {
		t.Fatalf("expected 'Last Known Good' in welcome screen, got %q", v)
	}
	if !contains(v, "The open source AI coding agent") {
		t.Fatalf("expected subtitle in welcome screen, got %q", v)
	}
}

func TestWelcomeScreenTransitionsToChatOnSend(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.input = "hello"
	m.sendMessage()

	if !m.started {
		t.Fatal("expected started to be true after send")
	}
}

func TestWelcomeScreenTransitionsToChatOnChunk(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:    agent.EventModelResponseChunk,
		Content: "Hello",
	})

	if !m.started {
		t.Fatal("expected started to be true after receiving a chunk")
	}
}

func TestChannelCloseTriggersQuit(t *testing.T) {
	events := make(chan agent.AgentEvent)
	submit := make(chan string, 10)
	m := New(events, submit)

	close(events)

	msg := m.waitForEvent()
	if msg != tea.Quit() {
		t.Fatalf("expected tea.Quit(), got %T", msg)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && containsStr(s, substr)
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
