package core

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessageConstruction(t *testing.T) {
	m := Message{
		Role:    "user",
		Content: "Hello",
	}
	if m.Role != "user" {
		t.Errorf("expected role 'user', got %q", m.Role)
	}
	if m.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", m.Content)
	}
	if m.ToolCalls != nil {
		t.Errorf("expected nil ToolCalls, got %v", m.ToolCalls)
	}
	if m.ToolResult != nil {
		t.Errorf("expected nil ToolResult, got %v", m.ToolResult)
	}
}

func TestMessageWithToolCall(t *testing.T) {
	m := Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "call_1", Name: "read_file", Arguments: `{"path":"test.txt"}`},
		},
	}
	if len(m.ToolCalls) != 1 {
		t.Fatalf("expected 1 ToolCall, got %d", len(m.ToolCalls))
	}
	if m.ToolCalls[0].ID != "call_1" {
		t.Errorf("expected ID 'call_1', got %q", m.ToolCalls[0].ID)
	}
	if m.ToolCalls[0].Name != "read_file" {
		t.Errorf("expected Name 'read_file', got %q", m.ToolCalls[0].Name)
	}
	if m.ToolCalls[0].Arguments != `{"path":"test.txt"}` {
		t.Errorf("expected Arguments %q, got %q", `{"path":"test.txt"}`, m.ToolCalls[0].Arguments)
	}
}

func TestMessageWithToolResult(t *testing.T) {
	m := Message{
		Role:    "tool",
		Content: "file contents",
		ToolResult: &ToolResult{
			ToolCallID: "call_1",
			Content:    "file contents",
			IsError:    false,
		},
	}
	if m.ToolResult == nil {
		t.Fatal("expected non-nil ToolResult")
	}
	if m.ToolResult.ToolCallID != "call_1" {
		t.Errorf("expected ToolCallID 'call_1', got %q", m.ToolResult.ToolCallID)
	}
	if m.ToolResult.Content != "file contents" {
		t.Errorf("expected Content 'file contents', got %q", m.ToolResult.Content)
	}
	if m.ToolResult.IsError {
		t.Error("expected IsError=false")
	}
}

func TestToolResultWithError(t *testing.T) {
	tr := ToolResult{
		ToolCallID: "call_2",
		Content:    "command not found",
		IsError:    true,
	}
	if !tr.IsError {
		t.Error("expected IsError=true")
	}
}

func TestToolResultWithMetadata(t *testing.T) {
	m := map[string]any{"exit_code": float64(1), "duration_ms": float64(42)}
	tr := ToolResult{
		ToolCallID: "call_3",
		Content:    "output",
		Metadata:   m,
	}
	if len(tr.Metadata) != 2 {
		t.Fatalf("expected 2 metadata entries, got %d", len(tr.Metadata))
	}
	if tr.Metadata["exit_code"] != float64(1) {
		t.Errorf("expected exit_code 1, got %v", tr.Metadata["exit_code"])
	}
}

func TestToolResultNilMetadata(t *testing.T) {
	tr := ToolResult{
		ToolCallID: "call_4",
		Content:    "output",
	}
	if tr.Metadata != nil {
		t.Error("expected nil Metadata by default")
	}
}

func TestResultAsNonChunk(t *testing.T) {
	r := Result{
		Content:   "final answer",
		ToolCalls: nil,
		IsChunk:   false,
		Done:      true,
		Err:       nil,
	}
	if r.IsChunk {
		t.Error("expected IsChunk=false")
	}
	if !r.Done {
		t.Error("expected Done=true")
	}
	if r.Err != nil {
		t.Errorf("expected nil error, got %v", r.Err)
	}
}

func TestResultAsChunk(t *testing.T) {
	r := Result{
		Content: "partial",
		IsChunk: true,
		Done:    false,
	}
	if !r.IsChunk {
		t.Error("expected IsChunk=true")
	}
	if r.Done {
		t.Error("expected Done=false")
	}
}

func TestResultWithError(t *testing.T) {
	r := Result{
		Err:  errTest,
		Done: true,
	}
	if r.Err == nil {
		t.Fatal("expected non-nil error")
	}
	if r.Err.Error() != "test error" {
		t.Errorf("expected 'test error', got %q", r.Err.Error())
	}
	if !r.Done {
		t.Error("expected Done=true on error")
	}
}

func TestResultWithToolCalls(t *testing.T) {
	r := Result{
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "call_1", Name: "bash", Arguments: `{"cmd":"ls"}`},
			{ID: "call_2", Name: "read_file", Arguments: `{"path":"x.txt"}`},
		},
		Done: true,
	}
	if len(r.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(r.ToolCalls))
	}
	if r.ToolCalls[0].Name != "bash" {
		t.Errorf("expected first tool 'bash', got %q", r.ToolCalls[0].Name)
	}
	if r.ToolCalls[1].Name != "read_file" {
		t.Errorf("expected second tool 'read_file', got %q", r.ToolCalls[1].Name)
	}
}

func TestMessageJSONRoundTrip(t *testing.T) {
	m := Message{
		Role:    "user",
		Content: "Hello",
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Role != "user" {
		t.Errorf("expected role 'user', got %q", decoded.Role)
	}
	if decoded.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", decoded.Content)
	}
}

func TestMessageJSONRoundTripWithToolCalls(t *testing.T) {
	m := Message{
		Role:    "assistant",
		Content: "",
		ToolCalls: []ToolCall{
			{ID: "call_x", Name: "grep", Arguments: `{"pattern":"foo","path":"bar.txt"}`},
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(decoded.ToolCalls))
	}
	if decoded.ToolCalls[0].ID != "call_x" {
		t.Errorf("expected ID 'call_x', got %q", decoded.ToolCalls[0].ID)
	}
	if decoded.ToolCalls[0].Name != "grep" {
		t.Errorf("expected Name 'grep', got %q", decoded.ToolCalls[0].Name)
	}
}

func TestMessageJSONRoundTripWithToolResult(t *testing.T) {
	m := Message{
		Role:    "tool",
		Content: "result content",
		ToolResult: &ToolResult{
			ToolCallID: "call_y",
			Content:    "result content",
			IsError:    true,
			Metadata:   map[string]any{"exit_code": float64(1)},
		},
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ToolResult == nil {
		t.Fatal("expected non-nil ToolResult")
	}
	if decoded.ToolResult.ToolCallID != "call_y" {
		t.Errorf("expected ToolCallID 'call_y', got %q", decoded.ToolResult.ToolCallID)
	}
	if !decoded.ToolResult.IsError {
		t.Error("expected IsError=true")
	}
	if decoded.ToolResult.Metadata == nil {
		t.Fatal("expected non-nil Metadata")
	}
	if decoded.ToolResult.Metadata["exit_code"] != float64(1) {
		t.Errorf("expected exit_code 1, got %v", decoded.ToolResult.Metadata["exit_code"])
	}
}

func TestToolCallJSONRoundTrip(t *testing.T) {
	tc := ToolCall{
		ID:        "call_42",
		Name:      "bash",
		Arguments: `{"cmd":"echo hi"}`,
	}
	data, err := json.Marshal(tc)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded ToolCall
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ID != "call_42" {
		t.Errorf("expected ID 'call_42', got %q", decoded.ID)
	}
	if decoded.Name != "bash" {
		t.Errorf("expected Name 'bash', got %q", decoded.Name)
	}
	if decoded.Arguments != `{"cmd":"echo hi"}` {
		t.Errorf("expected Arguments %q, got %q", `{"cmd":"echo hi"}`, decoded.Arguments)
	}
}

func TestToolResultJSONRoundTrip(t *testing.T) {
	tr := ToolResult{
		ToolCallID: "call_99",
		Content:    "some output",
		IsError:    false,
		Metadata:   map[string]any{"duration_ms": float64(150)},
	}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.ToolCallID != "call_99" {
		t.Errorf("expected ToolCallID 'call_99', got %q", decoded.ToolCallID)
	}
	if decoded.Content != "some output" {
		t.Errorf("expected Content 'some output', got %q", decoded.Content)
	}
	if decoded.IsError {
		t.Error("expected IsError=false")
	}
}

func TestToolResultJSONNullMetadata(t *testing.T) {
	tr := ToolResult{
		ToolCallID: "call_100",
		Content:    "no metadata",
	}
	data, err := json.Marshal(tr)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded ToolResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Metadata != nil {
		t.Error("expected nil Metadata after round-trip")
	}
}

func TestResultJSONRoundTrip(t *testing.T) {
	r := Result{
		Content:   "hello",
		ToolCalls: []ToolCall{{ID: "c1", Name: "bash", Arguments: `{}`}},
		IsChunk:   true,
		Done:      false,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var decoded Result
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if decoded.Content != "hello" {
		t.Errorf("expected Content 'hello', got %q", decoded.Content)
	}
	if len(decoded.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(decoded.ToolCalls))
	}
	if !decoded.IsChunk {
		t.Error("expected IsChunk=true")
	}
	if decoded.Done {
		t.Error("expected Done=false")
	}
}

func TestResultJSONMarshalErrorField(t *testing.T) {
	r := Result{Err: errTest}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	// Err is an interface{} at the JSON level; unmarshal into a generic map
	// to confirm the Err field serializes as its concrete value.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if _, ok := raw["Err"]; !ok {
		t.Error("expected Err field in JSON output")
	}
	if len(raw) != 5 {
		t.Errorf("expected 5 fields (Content, ToolCalls, IsChunk, Done, Err), got %d", len(raw))
	}
}

func TestZeroValues(t *testing.T) {
	var m Message
	if m.Role != "" {
		t.Errorf("expected empty Role, got %q", m.Role)
	}
	var tc ToolCall
	if tc.ID != "" {
		t.Errorf("expected empty ID, got %q", tc.ID)
	}
	var tr ToolResult
	if tr.ToolCallID != "" {
		t.Errorf("expected empty ToolCallID, got %q", tr.ToolCallID)
	}
	var r Result
	if r.Content != "" {
		t.Errorf("expected empty Content, got %q", r.Content)
	}
	if r.Err != nil {
		t.Errorf("expected nil Err, got %v", r.Err)
	}
}

func TestBuildSystemPromptReturnsPersona(t *testing.T) {
	result := BuildSystemPrompt("", "")
	if !strings.Contains(result, "Last Known Good") {
		t.Error("expected persona to contain 'Last Known Good'")
	}
	if !strings.Contains(result, "direct deadpan, sarcastic and witty") {
		t.Error("expected persona to contain 'direct deadpan, sarcastic and witty'")
	}
	if strings.Contains(result, "## Available Skills") {
		t.Error("expected no skills section when summaries are empty")
	}
	if strings.Contains(result, "## Available Tools") {
		t.Error("expected no tools section when descriptions are empty")
	}
}

func TestBuildSystemPromptIncludesSkills(t *testing.T) {
	skills := "- **code-review**: Review pull requests for common issues"
	result := BuildSystemPrompt(skills, "")
	if !strings.Contains(result, "## Available Skills") {
		t.Error("expected skills section heading")
	}
	if !strings.Contains(result, "code-review") {
		t.Error("expected skill summary in output")
	}
}

func TestBuildSystemPromptIncludesTools(t *testing.T) {
	tools := "- **read_file**: Read a file from the workspace"
	result := BuildSystemPrompt("", tools)
	if !strings.Contains(result, "## Available Tools") {
		t.Error("expected tools section heading")
	}
	if !strings.Contains(result, "read_file") {
		t.Error("expected tool description in output")
	}
}

var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }
