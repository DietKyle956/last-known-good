package hooks

import (
	"context"
	"fmt"
	"sync"

	"github.com/DietKyle956/last-known-good/internal/core"
)

// HookType categorises hook events in the session lifecycle.
type HookType int

const (
	SessionStarted   HookType = iota + 1
	SessionEnded
	BeforeModelCall
	AfterModelCall
	BeforeToolCall
	AfterToolCall
)

func (t HookType) String() string {
	switch t {
	case SessionStarted:
		return "session_started"
	case SessionEnded:
		return "session_ended"
	case BeforeModelCall:
		return "before_model_call"
	case AfterModelCall:
		return "after_model_call"
	case BeforeToolCall:
		return "before_tool_call"
	case AfterToolCall:
		return "after_tool_call"
	default:
		return fmt.Sprintf("unknown(%d)", int(t))
	}
}

// HookEvent carries data for a hook invocation.
type HookEvent struct {
	Type       HookType
	SessionID  int64
	Model      string
	ToolCall   *core.ToolCall
	ToolResult *core.ToolResult
	Error      error
}

// HookResult indicates whether a before-hook wants to block execution.
// Only BeforeToolCall hooks may return Block=true.
type HookResult struct {
	Block bool
}

// HookFunc processes a hook event.
type HookFunc func(HookEvent) *HookResult

// System manages hook registration and dispatch.
type System struct {
	mu     sync.Mutex
	hooks  map[HookType][]HookFunc
	events <-chan core.AgentEvent
	ctx    context.Context
	cancel context.CancelFunc
}

// New creates a hooks system that subscribes to the given agent event channel.
// Pass nil for events to create a system without channel subscription.
func New(events <-chan core.AgentEvent) *System {
	ctx, cancel := context.WithCancel(context.Background())
	s := &System{
		hooks:  make(map[HookType][]HookFunc),
		events: events,
		ctx:    ctx,
		cancel: cancel,
	}
	if events != nil {
		go s.consumeEvents()
	}
	return s
}

// Stop shuts down the event consumption goroutine.
func (s *System) Stop() {
	s.cancel()
}

// Register adds a hook function for the given event type.
func (s *System) Register(t HookType, fn HookFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hooks[t] = append(s.hooks[t], fn)
}

// Notify synchronously invokes all hooks registered for the event type.
// Hooks fire in registration order. For BeforeToolCall, returns true if
// any hook blocked execution. For all other types, Block is ignored.
func (s *System) Notify(ctx context.Context, event HookEvent) bool {
	s.mu.Lock()
	fns := make([]HookFunc, len(s.hooks[event.Type]))
	copy(fns, s.hooks[event.Type])
	s.mu.Unlock()

	blocked := false
	for _, fn := range fns {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		r := fn(event)
		if r != nil && r.Block && event.Type == BeforeToolCall {
			blocked = true
		}
	}
	return blocked
}

// consumeEvents reads from the agent event channel and dispatches
// corresponding hook events.
func (s *System) consumeEvents() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case ev, ok := <-s.events:
			if !ok {
				return
			}
			s.dispatchChannelEvent(ev)
		}
	}
}

func (s *System) dispatchChannelEvent(ev core.AgentEvent) {
	ctx := context.Background()
	switch ev.Type {
	case core.EventToolCallStarted:
		if ev.ToolCall != nil {
			s.Notify(ctx, HookEvent{Type: BeforeToolCall, ToolCall: ev.ToolCall})
		}
	case core.EventToolCallFinished:
		if ev.ToolCall != nil {
			s.Notify(ctx, HookEvent{Type: AfterToolCall, ToolCall: ev.ToolCall, ToolResult: ev.ToolResult})
		}
	case core.EventTurnComplete:
		s.Notify(ctx, HookEvent{Type: AfterModelCall})
	case core.EventError:
		s.Notify(ctx, HookEvent{Type: AfterModelCall, Error: ev.Error})
	}
}
