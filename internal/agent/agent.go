package agent

import (
	"fmt"
	"sync"
)

// Message represents a single message in the conversation.
type Message struct {
	Role        string
	Content     string
	ToolCalls   []ToolCall
	ToolResult  *ToolResult
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
}

// AgentEventType categorises agent events.
type AgentEventType int

const (
	EventModelResponseChunk AgentEventType = iota + 1
	EventToolCallStarted
	EventToolCallFinished
	EventTurnComplete
	EventError
)

// AgentEvent is emitted by the agent loop at lifecycle points.
type AgentEvent struct {
	Type       AgentEventType
	Content    string
	ToolCall   *ToolCall
	ToolResult *ToolResult
	Error      error
}

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

// Result is a single item from the LLM stream.
type Result struct {
	Content   string
	ToolCalls []ToolCall
	IsChunk   bool
	Done      bool
	Err       error
}

// LLM is the interface for calling a language model.
type LLM interface {
	Chat(messages []Message) (<-chan Result, error)
}

// ToolExecutor executes tool calls and provides metadata.
type ToolExecutor interface {
	Execute(call ToolCall) ToolResult
	IsReadOnly(name string) bool
}

// Agent runs the core agent loop.
type Agent struct {
	llm    LLM
	exec   ToolExecutor
	events chan AgentEvent
}

// New creates a new Agent.
func New(llm LLM, exec ToolExecutor) *Agent {
	return &Agent{
		llm:    llm,
		exec:   exec,
		events: make(chan AgentEvent, 64),
	}
}

// Events returns a read-only channel of agent events.
func (a *Agent) Events() <-chan AgentEvent {
	return a.events
}

// Run starts the agent loop with the given messages.
func (a *Agent) Run(messages []Message) {
	defer close(a.events)
	a.loop(messages)
}

func (a *Agent) loop(messages []Message) {
	results, err := a.llm.Chat(messages)
	if err != nil {
		a.events <- AgentEvent{Type: EventError, Error: err}
		return
	}

	var toolCalls []ToolCall

	for r := range results {
		if r.Err != nil {
			a.events <- AgentEvent{Type: EventError, Error: r.Err}
			return
		}
		if r.IsChunk {
			a.events <- AgentEvent{Type: EventModelResponseChunk, Content: r.Content}
		}
		if len(r.ToolCalls) > 0 {
			toolCalls = append(toolCalls, r.ToolCalls...)
		}
	}

	if len(toolCalls) > 0 {
		messages = a.executeToolCalls(messages, toolCalls)
		a.loop(messages)
		return
	}

	a.events <- AgentEvent{Type: EventTurnComplete}
}

func (a *Agent) executeToolCalls(messages []Message, calls []ToolCall) []Message {
	var ro, rw []ToolCall
	for _, tc := range calls {
		if a.exec.IsReadOnly(tc.Name) {
			ro = append(ro, tc)
		} else {
			rw = append(rw, tc)
		}
	}

	results := make([]ToolResult, len(calls))
	resultIdx := make(map[string]int, len(calls))
	for i, tc := range calls {
		resultIdx[tc.ID] = i
	}

	// Emit all started events first (synchronous — determinisitic ordering).
	for _, tc := range ro {
		a.events <- AgentEvent{Type: EventToolCallStarted, ToolCall: &tc}
	}
	for _, tc := range rw {
		a.events <- AgentEvent{Type: EventToolCallStarted, ToolCall: &tc}
	}

	// Execute read-only tools in parallel.
	if len(ro) > 0 {
		var wg sync.WaitGroup
		for _, tc := range ro {
			wg.Add(1)
			tc := tc
			go func() {
				defer wg.Done()
				r := a.exec.Execute(tc)
				a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
				results[resultIdx[tc.ID]] = r
			}()
		}
		wg.Wait()
	}

	// Execute write tools sequentially (one at a time).
	for _, tc := range rw {
		r := a.exec.Execute(tc)
		a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
		results[resultIdx[tc.ID]] = r
	}

	for _, r := range results {
		messages = append(messages, Message{
			Role:       "tool",
			ToolResult: &r,
		})
	}

	return messages
}

