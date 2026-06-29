package llm

import (
	"encoding/json"
	"testing"
)

func TestDeepSeekRequestJSONRoundTrip(t *testing.T) {
	req := DeepSeekRequest{
		Model: "deepseek-v4-pro",
		Messages: []DeepSeekMessage{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there"},
		},
		Stream: false,
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded DeepSeekRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Model != "deepseek-v4-pro" {
		t.Errorf("expected model 'deepseek-v4-pro', got %q", decoded.Model)
	}
	if len(decoded.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(decoded.Messages))
	}
	if decoded.Messages[0].Role != "user" || decoded.Messages[0].Content != "Hello" {
		t.Errorf("unexpected message 0: %+v", decoded.Messages[0])
	}
	if decoded.Stream {
		t.Error("expected Stream=false")
	}
}

func TestDeepSeekResponseParsesContent(t *testing.T) {
	body := `{"id":"chat-123","choices":[{"index":0,"message":{"role":"assistant","content":"Hello!"},"finish_reason":"stop"}]}`
	var resp DeepSeekResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello!" {
		t.Errorf("expected content 'Hello!', got %q", resp.Choices[0].Message.Content)
	}
	if resp.Choices[0].FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.Choices[0].FinishReason)
	}
}

func TestDeepSeekResponseParsesToolCalls(t *testing.T) {
	body := `{"id":"chat-456","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]},"finish_reason":"tool_calls"}]}`
	var resp DeepSeekResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	tcs := resp.Choices[0].Message.ToolCalls
	if len(tcs) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(tcs))
	}
	if tcs[0].ID != "call_1" {
		t.Errorf("expected ID 'call_1', got %q", tcs[0].ID)
	}
	if tcs[0].Function.Name != "read_file" {
		t.Errorf("expected function name 'read_file', got %q", tcs[0].Function.Name)
	}
	if tcs[0].Function.Arguments != `{"path":"test.txt"}` {
		t.Errorf("unexpected arguments: %q", tcs[0].Function.Arguments)
	}
}

func TestDeepSeekChunkParsesDeltaContent(t *testing.T) {
	body := `{"id":"chunk-1","choices":[{"index":0,"delta":{"content":"Hello"}}]}`
	var chunk DeepSeekChunk
	if err := json.Unmarshal([]byte(body), &chunk); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(chunk.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(chunk.Choices))
	}
	if chunk.Choices[0].Delta.Content != "Hello" {
		t.Errorf("expected content 'Hello', got %q", chunk.Choices[0].Delta.Content)
	}
}

func TestDeepSeekChunkWithToolCallDelta(t *testing.T) {
	body := `{"id":"chunk-2","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":"}}]}}]}`
	var chunk DeepSeekChunk
	if err := json.Unmarshal([]byte(body), &chunk); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(chunk.Choices) == 0 {
		t.Fatal("expected at least 1 choice")
	}
	if len(chunk.Choices[0].Delta.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call delta, got %d", len(chunk.Choices[0].Delta.ToolCalls))
	}
	if chunk.Choices[0].Delta.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("expected function name 'read_file', got %q", chunk.Choices[0].Delta.ToolCalls[0].Function.Name)
	}
}
