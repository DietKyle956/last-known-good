package singleshot

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
)

func TestTextRendererWritesChunksAndExitsOnTurnComplete(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	var buf bytes.Buffer

	r := New(events, &buf, false)

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run()
	}()

	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "Hello, "}
	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "world!"}
	events <- agent.AgentEvent{Type: agent.EventTurnComplete}

	err := <-errCh
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "Hello, world!"
	if buf.String() != expected {
		t.Fatalf("expected %q, got %q", expected, buf.String())
	}
}

func TestJSONRendererOutputsStructuredJSON(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	var buf bytes.Buffer

	r := New(events, &buf, true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run()
	}()

	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "{\"result\": "}
	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "\"hello\"}"}
	events <- agent.AgentEvent{Type: agent.EventToolCallStarted, ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`}}
	events <- agent.AgentEvent{Type: agent.EventToolCallFinished, ToolCall: &core.ToolCall{ID: "c1", Name: "read_file", Arguments: `{"path":"x.txt"}`}, ToolResult: &core.ToolResult{ToolCallID: "c1", Content: "file content"}}
	events <- agent.AgentEvent{Type: agent.EventTurnComplete}

	err := <-errCh
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, buf.String())
	}

	if out["success"] != true {
		t.Fatal("expected success=true")
	}
	if !strings.Contains(out["content"].(string), "hello") {
		t.Fatalf("expected content to contain 'hello', got %v", out["content"])
	}
	tools, ok := out["tool_calls"].([]any)
	if !ok || len(tools) == 0 {
		t.Fatal("expected tool_calls array with entries")
	}
	tc := tools[0].(map[string]any)
	if tc["name"] != "read_file" {
		t.Fatalf("expected tool name 'read_file', got %v", tc["name"])
	}
}

func TestTextRendererReturnsErrorOnAgentError(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	var buf bytes.Buffer

	r := New(events, &buf, false)

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run()
	}()

	events <- agent.AgentEvent{Type: agent.EventError, Error: errors.New("oops")}

	err := <-errCh
	if err == nil || err.Error() != "oops" {
		t.Fatalf("expected error 'oops', got %v", err)
	}
}

func TestJSONRendererSetsSuccessFalseOnError(t *testing.T) {
	events := make(chan agent.AgentEvent, 10)
	var buf bytes.Buffer

	r := New(events, &buf, true)

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.Run()
	}()

	events <- agent.AgentEvent{Type: agent.EventModelResponseChunk, Content: "partial "}
	events <- agent.AgentEvent{Type: agent.EventError, Error: errors.New("something broke")}

	err := <-errCh
	if err == nil {
		t.Fatal("expected Run to return error on agent error")
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if out["success"] != false {
		t.Fatal("expected success=false on error")
	}
	if !strings.Contains(out["content"].(string), "something broke") {
		t.Fatalf("expected content to contain error message, got %v", out["content"])
	}
}
