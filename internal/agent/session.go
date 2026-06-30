package agent

import (
	"context"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
	"github.com/DietKyle956/last-known-good/internal/router"
)

// Session manages a multi-turn agent conversation.
type Session struct {
	llm        LLM
	exec       ToolExecutor
	router     router.Router
	llmFactory func(router.RouteDecision) LLM
	hooks      *hooks.System
}

// SetHooks attaches a hooks system to this session.
func (s *Session) SetHooks(h *hooks.System) {
	s.hooks = h
}

// NewSession creates a new Session with a fixed LLM.
func NewSession(llm LLM, exec ToolExecutor) *Session {
	return &Session{llm: llm, exec: exec}
}

// NewSessionWithRouter creates a Session that uses a Router to select
// the LLM per turn. The llmFactory receives each routing decision and
// must return an LLM configured accordingly.
func NewSessionWithRouter(exec ToolExecutor, r router.Router, llmFactory func(router.RouteDecision) LLM) *Session {
	return &Session{exec: exec, router: r, llmFactory: llmFactory}
}

// Run starts the multi-turn loop. For each turn:
//  1. If a Router is configured, uses it to select the LLM for this turn
//  2. Creates a new Agent
//  3. Runs it with the accumulated messages
//  4. Forwards events to the events channel
//  5. Waits for the next prompt on submit
//  6. Appends the user message and loops
//
// Closes events when submit is closed or the context is done.
func (s *Session) Run(ctx context.Context, messages []core.Message, submit <-chan string, events chan<- AgentEvent) {
	defer close(events)

	if s.hooks != nil {
		s.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.SessionStarted})
		defer s.hooks.Notify(ctx, hooks.HookEvent{Type: hooks.SessionEnded})
	}

	var prevFailed bool
	var prevTouchedFiles int

	for {
		turnLLM := s.llm
		if s.router != nil {
			prompt := ""
			if len(messages) > 0 {
				prompt = messages[len(messages)-1].Content
			}
			dec := s.router.Route(ctx, router.RouteRequest{
				TouchedFiles: prevTouchedFiles,
				Failed:       prevFailed,
				Prompt:       prompt,
			})
			turnLLM = s.llmFactory(dec)
		}

		a := New(turnLLM, s.exec)
		if s.hooks != nil {
			a.SetHooks(s.hooks)
		}

		forwardDone := make(chan struct{})
		go func() {
			defer close(forwardDone)
			for ev := range a.Events() {
				select {
				case events <- ev:
				case <-ctx.Done():
					return
				}
			}
		}()

		a.Run(ctx, messages)

		<-forwardDone

		select {
		case prompt, ok := <-submit:
			if !ok || prompt == "" {
				return
			}
			messages = append(messages, core.Message{Role: "user", Content: prompt})
		case <-ctx.Done():
			return
		}
	}
}
