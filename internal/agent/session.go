package agent

import (
	"context"

	"github.com/DietKyle956/last-known-good/internal/core"
)

// Session manages a multi-turn agent conversation.
type Session struct {
	llm  LLM
	exec ToolExecutor
}

// NewSession creates a new Session.
func NewSession(llm LLM, exec ToolExecutor) *Session {
	return &Session{llm: llm, exec: exec}
}

// Run starts the multi-turn loop. For each turn:
//  1. Creates a new Agent
//  2. Runs it with the accumulated messages
//  3. Forwards events to the events channel
//  4. Waits for the next prompt on submit
//  5. Appends the user message and loops
//
// Closes events when submit is closed or the context is done.
func (s *Session) Run(ctx context.Context, messages []core.Message, submit <-chan string, events chan<- AgentEvent) {
	defer close(events)
	for {
		a := New(s.llm, s.exec)

		// Forward agent events to the session events channel.
		// Track completion so the loop doesn't race ahead.
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

		// Wait for all events to be forwarded before checking submit,
		// preventing a race where the session exits before events arrive.
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
