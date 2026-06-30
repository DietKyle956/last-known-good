package core

import "fmt"

// AgentEventType categorises agent events.
type AgentEventType int

const (
	EventModelResponseChunk AgentEventType = iota + 1
	EventToolCallStarted
	EventToolCallFinished
	EventTurnComplete
	EventError
)

func (t AgentEventType) String() string {
	switch t {
	case EventModelResponseChunk:
		return "model_response_chunk"
	case EventToolCallStarted:
		return "tool_call_started"
	case EventToolCallFinished:
		return "tool_call_finished"
	case EventTurnComplete:
		return "turn_complete"
	case EventError:
		return "error"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// AgentEvent is emitted by the agent loop at lifecycle points.
type AgentEvent struct {
	Type       AgentEventType
	Content    string
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Error      error
}

// Message represents a single message in the conversation.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolResult *ToolResult
}

// ToolCall represents a tool call requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
	Metadata   map[string]any
}

// Result is a single item from the LLM stream.
type Result struct {
	Content   string
	ToolCalls []ToolCall
	IsChunk   bool
	Done      bool
	Err       error
}
