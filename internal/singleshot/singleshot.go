package singleshot

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/DietKyle956/last-known-good/internal/agent"
)

type Renderer struct {
	events <-chan agent.AgentEvent
	w      io.Writer
	json   bool
}

func New(events <-chan agent.AgentEvent, w io.Writer, json bool) *Renderer {
	return &Renderer{events: events, w: w, json: json}
}

func (r *Renderer) Run() error {
	if r.json {
		return r.runJSON()
	}
	return r.runText()
}

func (r *Renderer) runText() error {
	for ev := range r.events {
		switch ev.Type {
		case agent.EventModelResponseChunk:
			fmt.Fprint(r.w, ev.Content)
		case agent.EventTurnComplete:
			return nil
		case agent.EventError:
			if ev.Error != nil {
				return ev.Error
			}
			return fmt.Errorf("agent error")
		}
	}
	return nil
}

type jsonToolCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Result    string `json:"result"`
	IsError   bool   `json:"is_error"`
}

type jsonOutput struct {
	Content   string         `json:"content"`
	Success   bool           `json:"success"`
	ToolCalls []jsonToolCall `json:"tool_calls,omitempty"`
}

func (r *Renderer) runJSON() error {
	var out jsonOutput
	out.Success = true

	for ev := range r.events {
		switch ev.Type {
		case agent.EventModelResponseChunk:
			out.Content += ev.Content
		case agent.EventToolCallFinished:
			if ev.ToolCall != nil {
				tc := jsonToolCall{
					Name:      ev.ToolCall.Name,
					Arguments: ev.ToolCall.Arguments,
				}
				if ev.ToolResult != nil {
					tc.Result = ev.ToolResult.Content
					tc.IsError = ev.ToolResult.IsError
				}
				out.ToolCalls = append(out.ToolCalls, tc)
			}
		case agent.EventTurnComplete:
			return json.NewEncoder(r.w).Encode(out)
		case agent.EventError:
			out.Success = false
			if ev.Error != nil {
				out.Content += fmt.Sprintf("error: %v", ev.Error)
			}
			if err := json.NewEncoder(r.w).Encode(out); err != nil {
				return err
			}
			return fmt.Errorf("agent error: %w", ev.Error)
		}
	}
	return json.NewEncoder(r.w).Encode(out)
}
