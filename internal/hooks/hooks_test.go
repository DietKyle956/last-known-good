package hooks

import (
	"context"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
)

func TestSessionStartedHookFires(t *testing.T) {
	s := New(nil)
	var fired bool
	s.Register(SessionStarted, func(HookEvent) *HookResult {
		fired = true
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: SessionStarted})
	if !fired {
		t.Fatal("expected SessionStarted hook to fire")
	}
}

func TestBeforeModelCallHookFires(t *testing.T) {
	s := New(nil)
	var fired bool
	s.Register(BeforeModelCall, func(HookEvent) *HookResult {
		fired = true
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: BeforeModelCall})
	if !fired {
		t.Fatal("expected BeforeModelCall hook to fire")
	}
}

func TestBeforeToolCallHookCanBlock(t *testing.T) {
	s := New(nil)
	s.Register(BeforeToolCall, func(HookEvent) *HookResult {
		return &HookResult{Block: true, Reason: "test block"}
	})
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall})
	if r == nil {
		t.Fatal("expected Notify to return a result when a BeforeToolCall hook blocks")
	}
	if !r.Block {
		t.Fatal("expected result to have Block=true")
	}
	if r.Reason != "test block" {
		t.Fatalf("expected reason %q, got %q", "test block", r.Reason)
	}
}

func TestAfterToolCallHookFires(t *testing.T) {
	s := New(nil)
	var fired bool
	s.Register(AfterToolCall, func(HookEvent) *HookResult {
		fired = true
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall})
	if !fired {
		t.Fatal("expected AfterToolCall hook to fire")
	}
}

func TestAfterToolCallHookBlockIgnored(t *testing.T) {
	s := New(nil)
	s.Register(AfterToolCall, func(HookEvent) *HookResult {
		return &HookResult{Block: true, Reason: "should be ignored"}
	})
	r := s.Notify(context.Background(), HookEvent{Type: AfterToolCall})
	if r != nil {
		t.Fatal("expected AfterToolCall hook block to be ignored - only BeforeToolCall should block")
	}
}

func TestChannelEventsTriggerHooks(t *testing.T) {
	events := make(chan core.AgentEvent, 10)
	s := New(events)

	done := make(chan struct{})
	var beforeTool, afterTool, afterModel bool
	s.Register(BeforeToolCall, func(HookEvent) *HookResult {
		beforeTool = true
		return nil
	})
	s.Register(AfterToolCall, func(HookEvent) *HookResult {
		afterTool = true
		return nil
	})
	s.Register(AfterModelCall, func(HookEvent) *HookResult {
		afterModel = true
		if beforeTool && afterTool {
			close(done)
		}
		return nil
	})

	events <- core.AgentEvent{Type: core.EventToolCallStarted, ToolCall: &core.ToolCall{ID: "c1"}}
	events <- core.AgentEvent{Type: core.EventToolCallFinished, ToolCall: &core.ToolCall{ID: "c1"}, ToolResult: &core.ToolResult{ToolCallID: "c1"}}
	events <- core.AgentEvent{Type: core.EventTurnComplete}

	<-done
	s.Stop()

	if !beforeTool {
		t.Error("expected BeforeToolCall hook to fire from channel event")
	}
	if !afterTool {
		t.Error("expected AfterToolCall hook to fire from channel event")
	}
	if !afterModel {
		t.Error("expected AfterModelCall hook to fire from channel event")
	}
}

func TestMultipleHooksFireInRegistrationOrder(t *testing.T) {
	s := New(nil)
	var order []int
	s.Register(BeforeToolCall, func(HookEvent) *HookResult {
		order = append(order, 1)
		return nil
	})
	s.Register(BeforeToolCall, func(HookEvent) *HookResult {
		order = append(order, 2)
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: BeforeToolCall})
	if len(order) != 2 || order[0] != 1 || order[1] != 2 {
		t.Fatalf("expected hooks to fire in order [1,2], got %v", order)
	}
}

func TestBeforeToolCallHookDoesNotBlockByDefault(t *testing.T) {
	s := New(nil)
	s.Register(BeforeToolCall, func(HookEvent) *HookResult {
		return nil
	})
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall})
	if r != nil {
		t.Fatal("expected Notify to return nil when no hook blocks")
	}
}

func TestAfterModelCallHookFires(t *testing.T) {
	s := New(nil)
	var fired bool
	s.Register(AfterModelCall, func(HookEvent) *HookResult {
		fired = true
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: AfterModelCall})
	if !fired {
		t.Fatal("expected AfterModelCall hook to fire")
	}
}

func TestSessionEndedHookFires(t *testing.T) {
	s := New(nil)
	var fired bool
	s.Register(SessionEnded, func(HookEvent) *HookResult {
		fired = true
		return nil
	})
	s.Notify(context.Background(), HookEvent{Type: SessionEnded})
	if !fired {
		t.Fatal("expected SessionEnded hook to fire")
	}
}
