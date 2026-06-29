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

func TestModelRendersInputBar(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	// Simulate a window resize to initialize viewport
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

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages (label + content), got %d", len(m.messages))
	}
	if m.messages[0].content != "Assistant" {
		t.Fatalf("expected label 'Assistant', got %q", m.messages[0].content)
	}
	if m.messages[1].content != "Hello" {
		t.Fatalf("expected content 'Hello', got %q", m.messages[1].content)
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

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages (label + content), got %d", len(m.messages))
	}
	if m.messages[0].content != "Assistant" {
		t.Fatalf("expected label 'Assistant', got %q", m.messages[0].content)
	}
	if m.messages[1].content != "Hello world" {
		t.Fatalf("expected content 'Hello world', got %q", m.messages[1].content)
	}
}

func TestEventsChannelConsumedByModel(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Simulate sending an event through the channel the way the agent would
	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "Streaming "}
	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "text"}

	// The model processes them via waitForEvent/agentEventMsg
	// Simulate the Bubble Tea loop: call waitForEvent, then handle the message
	msg1 := m.waitForEvent()
	m.Update(msg1)
	msg2 := m.waitForEvent()
	m.Update(msg2)

	if len(m.messages) != 2 {
		t.Fatalf("expected 2 messages (label + content), got %d", len(m.messages))
	}
	if m.messages[0].content != "Assistant" {
		t.Fatalf("expected label 'Assistant', got %q", m.messages[0].content)
	}
	if m.messages[1].content != "Streaming text" {
		t.Fatalf("expected 'Streaming text', got %q", m.messages[1].content)
	}
}

func TestSubmitChannelSendsPrompt(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Type some text and press enter
	for _, r := range "hello" {
		m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEnter})

	select {
	case prompt := <-submit:
		if prompt != "hello" {
			t.Fatalf("expected 'hello', got %q", prompt)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for prompt on submit channel")
	}

	// User message should be rendered in the viewport with thinking indicator
	if len(m.messages) != 3 {
		t.Fatalf("expected 3 messages (label + content + thinking), got %d", len(m.messages))
	}
	if m.messages[0].content != "You" {
		t.Fatalf("expected label 'You', got %q", m.messages[0].content)
	}
	if m.messages[1].content != "hello" {
		t.Fatalf("expected content 'hello', got %q", m.messages[1].content)
	}
	if m.messages[2].content != "…" {
		t.Fatalf("expected thinking indicator '…', got %q", m.messages[2].content)
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

	if len(m.messages) < 2 {
		t.Fatalf("expected at least 2 messages, got %d", len(m.messages))
	}
	if !contains(m.messages[0].content, "read_file") {
		t.Fatalf("expected tool name in first line, got %q", m.messages[0].content)
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

	lastContent := m.messages[len(m.messages)-1].content
	if lastContent != "  … running" {
		t.Fatalf("expected '  … running', got %q", lastContent)
	}

	m.handleEvent(agent.AgentEvent{
		Type:     agent.EventToolCallFinished,
		ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`},
		ToolResult: &core.ToolResult{
			ToolCallID: "c1",
			Content:    "file contents",
			IsError:    false,
		},
	})

	// Should have the result line now, running line should be gone
	lastContent = m.messages[len(m.messages)-1].content
	if lastContent != "  → file contents" {
		t.Fatalf("expected '  → file contents', got %q", lastContent)
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

	lastContent := m.messages[len(m.messages)-1].content
	if !contains(lastContent, "permission denied") {
		t.Fatalf("expected error message in last line, got %q", lastContent)
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

	if len(m.messages) != 1 {
		t.Fatalf("expected 1 error message, got %d", len(m.messages))
	}
	if !contains(m.messages[0].content, "something went wrong") {
		t.Fatalf("expected error in message, got %q", m.messages[0].content)
	}
}

func TestTextWrapsAtViewportWidth(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	submit := make(chan string, 10)
	m := New(events, submit)

	// Set a narrow viewport to force wrapping
	m.Update(tea.WindowSizeMsg{Width: 30, Height: 24})

	m.handleEvent(agent.AgentEvent{
		Type:    agent.EventModelResponseChunk,
		Content: "This is a very long piece of text that should definitely be wrapped at the viewport width",
	})

	rendered := m.renderMessages()
	if !contains(rendered, "\n") {
		t.Fatalf("expected wrapped text (with newlines), got single line: %q", rendered)
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
