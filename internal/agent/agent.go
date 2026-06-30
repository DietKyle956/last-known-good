package agent

import (
	"context"
	"sync"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
)

// AgentEvent is shorthand for core.AgentEvent.
type AgentEvent = core.AgentEvent

// AgentEventType is shorthand for core.AgentEventType.
type AgentEventType = core.AgentEventType

// Convenience aliases for event type constants.
const (
	EventModelResponseChunk = core.EventModelResponseChunk
	EventToolCallStarted    = core.EventToolCallStarted
	EventToolCallFinished   = core.EventToolCallFinished
	EventTurnComplete       = core.EventTurnComplete
	EventError              = core.EventError
)

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
	hooks  *hooks.System
}

// New creates a new Agent.
func New(llm LLM, exec ToolExecutor) *Agent {
	return &Agent{
		llm:    llm,
		exec:   exec,
		events: make(chan AgentEvent, 64),
	}
}

// SetHooks attaches a hooks system to this agent.
func (a *Agent) SetHooks(h *hooks.System) {
	a.hooks = h
}

// Events returns a read-only channel of agent events.
func (a *Agent) Events() <-chan AgentEvent {
	return a.events
}

// Run starts the agent loop with the given messages.
func (a *Agent) Run(ctx context.Context, messages []core.Message) {
	defer close(a.events)
	for {
		if a.hooks != nil {
			a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.BeforeModelCall})
		}

		results, err := a.llm.Chat(ctx, messages)
		if err != nil {
			a.events <- AgentEvent{Type: EventError, Error: err}
			if a.hooks != nil {
				a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.AfterModelCall, Error: err})
			}
			return
		}

		var toolCalls []core.ToolCall

		for r := range results {
			if r.Err != nil {
				a.events <- AgentEvent{Type: EventError, Error: r.Err}
				if a.hooks != nil {
					a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.AfterModelCall, Error: r.Err})
				}
				return
			}
			if r.IsChunk {
				a.events <- AgentEvent{Type: EventModelResponseChunk, Content: r.Content}
			}
			if len(r.ToolCalls) > 0 {
				toolCalls = append(toolCalls, r.ToolCalls...)
			}
		}

		if a.hooks != nil {
			a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.AfterModelCall})
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

	// Check before-tool-call hooks and filter out blocked tools.
	var activeRO, activeRW []core.ToolCall
	for _, tc := range ro {
		if a.hooks != nil {
			if r := a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.BeforeToolCall, ToolCall: &tc}); r != nil {
				reason := r.Reason
				if reason == "" {
					reason = "blocked by hook"
				}
				results[resultIdx[tc.ID]] = core.ToolResult{ToolCallID: tc.ID, Content: reason, IsError: true}
				continue
			}
		}
		activeRO = append(activeRO, tc)
	}
	for _, tc := range rw {
		if a.hooks != nil {
			if r := a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.BeforeToolCall, ToolCall: &tc}); r != nil {
				reason := r.Reason
				if reason == "" {
					reason = "blocked by hook"
				}
				results[resultIdx[tc.ID]] = core.ToolResult{ToolCallID: tc.ID, Content: reason, IsError: true}
				continue
			}
		}
		activeRW = append(activeRW, tc)
	}

	// Emit all started events first (synchronous — determinisitic ordering).
	for _, tc := range activeRO {
		a.events <- AgentEvent{Type: EventToolCallStarted, ToolCall: &tc}
	}
	for _, tc := range activeRW {
		a.events <- AgentEvent{Type: EventToolCallStarted, ToolCall: &tc}
	}

	// Execute read-only tools in parallel.
	if len(activeRO) > 0 {
		var wg sync.WaitGroup
		for _, tc := range activeRO {
			wg.Add(1)
			tc := tc
			go func() {
				defer wg.Done()
				r := a.exec.Execute(ctx, tc)
				a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
				results[resultIdx[tc.ID]] = r
				if a.hooks != nil {
					a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.AfterToolCall, ToolCall: &tc, ToolResult: &r})
				}
			}()
		}
		wg.Wait()
	}

	// Execute write tools sequentially (one at a time).
	for _, tc := range activeRW {
		r := a.exec.Execute(ctx, tc)
		a.events <- AgentEvent{Type: EventToolCallFinished, ToolCall: &tc, ToolResult: &r}
		results[resultIdx[tc.ID]] = r
		if a.hooks != nil {
			a.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.AfterToolCall, ToolCall: &tc, ToolResult: &r})
		}
	}

	for _, r := range results {
		messages = append(messages, core.Message{
			Role:       "tool",
			ToolResult: &r,
		})
	}

	return messages
}
