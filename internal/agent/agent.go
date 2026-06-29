package agent

import (
	"context"
	"fmt"
	"sync"

	"github.com/DietKyle956/last-known-good/internal/core"
)

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
	ToolCall   *core.ToolCall
	ToolResult *core.ToolResult
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

// LLM is the interface for calling a language model.
type LLM interface {
	Chat(ctx context.Context, messages []core.Message) (<-chan core.Result, error)
}

// ToolExecutor executes tool calls and provides metadata.
type ToolExecutor interface {
	Execute(ctx context.Context, call core.ToolCall) core.ToolResult
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
func (a *Agent) Run(ctx context.Context, messages []core.Message) {
	defer close(a.events)
	for {
		results, err := a.llm.Chat(ctx, messages)
		if err != nil {
			a.events <- AgentEvent{Type: EventError, Error: err}
			return
		}

		var toolCalls []core.ToolCall

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

		if len(toolCalls) == 0 {
			a.events <- AgentEvent{Type: EventTurnComplete}
			return
		}

		messages = a.executeToolCalls(ctx, messages, toolCalls)
	}
}

func (a *Agent) executeToolCalls(ctx context.Context, messages []core.Message, calls []core.ToolCall) []core.Message {
	var ro, rw []core.ToolCall
	for _, tc := range calls {
		if a.exec.IsReadOnly(tc.Name) {
			ro = append(ro, tc)
		} else {
			rw = append(rw, tc)
		}
	}

	results := make([]core.ToolResult, len(calls))
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
				r := a.exec.Execute(ctx, tc)
				a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
				results[resultIdx[tc.ID]] = r
			}()
		}
		wg.Wait()
	}

	// Execute write tools sequentially (one at a time).
	for _, tc := range rw {
		r := a.exec.Execute(ctx, tc)
		a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
		results[resultIdx[tc.ID]] = r
	}

	for _, r := range results {
		messages = append(messages, core.Message{
			Role:       "tool",
			ToolResult: &r,
		})
	}

	return messages
}
