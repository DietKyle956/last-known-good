package agent

import (
	"context"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/router"
)

func TestSessionSingleTurn(t *testing.T) {
	llm := &noToolLLM{response: "Hello"}
	exec := &spyExecutor{}
	sess := NewSession(llm, exec)

	submit := make(chan string, 1)
	events := make(chan AgentEvent, 64)
	ctx := context.Background()
	go sess.Run(ctx, []core.Message{{Role: "user", Content: "Hi"}}, submit, events)

	// Close submit so the session exits after the first turn completes.
	close(submit)

	var got []AgentEvent
	for ev := range events {
		got = append(got, ev)
	}
	if len(got) == 0 {
		t.Fatal("expected at least one event")
	}
	if got[len(got)-1].Type != EventTurnComplete {
		t.Fatalf("expected last event to be TurnComplete, got %v", got[len(got)-1].Type)
	}
}

func TestSessionMultiTurn(t *testing.T) {
	llm := &noToolLLM{response: "Hello"}
	exec := &spyExecutor{}
	sess := NewSession(llm, exec)

	submit := make(chan string, 2)
	events := make(chan AgentEvent, 128)
	ctx := context.Background()

	submit <- "First prompt"
	submit <- "Second prompt"
	close(submit) // Close so the session exits after consuming both prompts.

	go sess.Run(ctx, []core.Message{{Role: "user", Content: "Initial"}}, submit, events)

	var turnCount int
	for ev := range events {
		if ev.Type == EventTurnComplete {
			turnCount++
		}
	}
	if turnCount != 3 { // initial + first prompt + second prompt
		t.Fatalf("expected 3 turns, got %d", turnCount)
	}
}

func TestSessionEmptyPromptCloses(t *testing.T) {
	llm := &noToolLLM{response: "Hello"}
	exec := &spyExecutor{}
	sess := NewSession(llm, exec)

	submit := make(chan string, 1)
	events := make(chan AgentEvent, 64)
	ctx := context.Background()

	submit <- "" // Empty prompt should close

	go sess.Run(ctx, []core.Message{{Role: "user", Content: "Hi"}}, submit, events)

	// Should close immediately after consuming the empty prompt
	for range events {
	}
}

func TestSessionContextCancellation(t *testing.T) {
	llm := &noToolLLM{response: "Hello"}
	exec := &spyExecutor{}
	sess := NewSession(llm, exec)

	submit := make(chan string, 1)
	events := make(chan AgentEvent, 64)
	ctx, cancel := context.WithCancel(context.Background())

	go sess.Run(ctx, []core.Message{{Role: "user", Content: "Hi"}}, submit, events)

	cancel()

	for range events {
	}
}

// recordingLLM wraps an LLM and records which model it was configured with.
type recordingLLM struct {
	LLM
	model    string
	thinking bool
}

func TestSessionWithRouterSelectsLLMPerTurn(t *testing.T) {
	exec := &spyExecutor{}

	var decs []router.RouteDecision
	routeCount := 0
	r := &router.HeuristicRouter{
		MultiFileThreshold: 2,
		ComplexitySignals:  router.DefaultSignals(),
	}

	llmFactory := func(dec router.RouteDecision) LLM {
		decs = append(decs, dec)
		routeCount++
		return &noToolLLM{response: "Hello"}
	}

	sess := NewSessionWithRouter(exec, r, llmFactory)

	submit := make(chan string, 2)
	events := make(chan AgentEvent, 128)
	ctx := context.Background()

	submit <- "First turn"
	close(submit)

	go sess.Run(ctx, []core.Message{{Role: "user", Content: "Hi"}}, submit, events)

	for range events {
	}

	if routeCount == 0 {
		t.Fatal("expected at least one routing decision")
	}
	// First turn: no prior turn info, single initial message — should be Flash
	if decs[0].Model != router.DeepSeekV4Flash {
		t.Errorf("expected first turn to use Flash, got %q", decs[0].Model)
	}
}
