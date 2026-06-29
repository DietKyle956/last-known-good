package llm

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
)

func TestDeepSeekClientNonStreamingReturnsContent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-1","choices":[{"index":0,"message":{"role":"assistant","content":"Hello, world!"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-v4-pro",
		BaseURL: srv.URL,
	})

	results, err := client.Chat([]core.Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var gotContent string
	var done bool
	for r := range results {
		if r.Err != nil {
			t.Fatalf("result error: %v", r.Err)
		}
		if r.IsChunk {
			gotContent += r.Content
		}
		if r.Done {
			done = true
		}
	}
	if !done {
		t.Fatal("expected Done result")
	}
	if gotContent != "Hello, world!" {
		t.Errorf("expected content 'Hello, world!', got %q", gotContent)
	}
}

func TestDeepSeekClientNonStreamingReturnsToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-2","choices":[{"index":0,"message":{"role":"assistant","content":"","tool_calls":[{"id":"call_1","type":"function","function":{"name":"read_file","arguments":"{\"path\":\"test.txt\"}"}}]},"finish_reason":"tool_calls"}]}`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-v4-flash",
		BaseURL: srv.URL,
	})

	results, err := client.Chat([]core.Message{{Role: "user", Content: "Read file"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var toolCalls []core.ToolCall
	for r := range results {
		if r.Err != nil {
			t.Fatalf("result error: %v", r.Err)
		}
		toolCalls = append(toolCalls, r.ToolCalls...)
	}
	if len(toolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(toolCalls))
	}
	if toolCalls[0].Name != "read_file" {
		t.Errorf("expected name 'read_file', got %q", toolCalls[0].Name)
	}
	if toolCalls[0].Arguments != `{"path":"test.txt"}` {
		t.Errorf("unexpected arguments: %q", toolCalls[0].Arguments)
	}
	if toolCalls[0].ID != "call_1" {
		t.Errorf("expected ID 'call_1', got %q", toolCalls[0].ID)
	}
}

func TestDeepSeekClientStreamingYieldsChunks(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"id\":\"chunk-1\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"Hel\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"id\":\"chunk-2\",\"choices\":[{\"index\":0,\"delta\":{\"content\":\"lo\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-v4-flash",
		BaseURL: srv.URL,
		Stream:  true,
	})

	results, err := client.Chat([]core.Message{{Role: "user", Content: "Say hi"}})
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var chunks []string
	var done bool
	for r := range results {
		if r.Err != nil {
			t.Fatalf("result error: %v", r.Err)
		}
		if r.IsChunk {
			chunks = append(chunks, r.Content)
		}
		if r.Done {
			done = true
		}
	}
	if !done {
		t.Fatal("expected Done result")
	}
	if len(chunks) != 2 {
		t.Fatalf("expected 2 chunks, got %d: %v", len(chunks), chunks)
	}
	if chunks[0] != "Hel" || chunks[1] != "lo" {
		t.Fatalf("unexpected chunks: %v", chunks)
	}
}

func TestDeepSeekClientRequestIncludesThinkingMode(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:       "test-key",
		Model:        "deepseek-v4-pro",
		BaseURL:      srv.URL,
		ThinkingMode: true,
	})

	results, _ := client.Chat([]core.Message{{Role: "user", Content: "Hi"}})
	for range results {
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}

	thinking, ok := req["thinking"].(map[string]any)
	if !ok {
		t.Fatal("expected 'thinking' object in request body")
	}
	if thinking["type"] != "enabled" {
		t.Errorf("expected thinking.type 'enabled', got %v", thinking["type"])
	}
}

func TestDeepSeekClientRequestIncludesReasoningEffort(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:          "test-key",
		Model:           "deepseek-v4-pro",
		BaseURL:         srv.URL,
		ReasoningEffort: "high",
	})

	results, _ := client.Chat([]core.Message{{Role: "user", Content: "Hi"}})
	for range results {
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}

	if req["reasoning_effort"] != "high" {
		t.Errorf("expected reasoning_effort 'high', got %v", req["reasoning_effort"])
	}
}

func TestDeepSeekClientRequestPayloadShape(t *testing.T) {
	var capturedBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"chat-1","choices":[{"index":0,"message":{"role":"assistant","content":"ok"},"finish_reason":"stop"}]}`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:       "test-key",
		Model:        "deepseek-v4-flash",
		BaseURL:      srv.URL,
		ThinkingMode: true,
	})

	results, _ := client.Chat([]core.Message{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello!"},
	})
	for range results {
	}

	var req map[string]any
	if err := json.Unmarshal(capturedBody, &req); err != nil {
		t.Fatalf("unmarshal captured body: %v", err)
	}

	if req["model"] != "deepseek-v4-flash" {
		t.Errorf("expected model 'deepseek-v4-flash', got %v", req["model"])
	}

	msgs, ok := req["messages"].([]any)
	if !ok || len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	msg0 := msgs[0].(map[string]any)
	if msg0["role"] != "system" || msg0["content"] != "You are a helpful assistant." {
		t.Errorf("unexpected message 0: %+v", msg0)
	}

	msg1 := msgs[1].(map[string]any)
	if msg1["role"] != "user" || msg1["content"] != "Hello!" {
		t.Errorf("unexpected message 1: %+v", msg1)
	}

	if _, exists := req["stream"]; !exists {
		t.Error("expected 'stream' field in request")
	}
	if req["stream"] != false {
		t.Errorf("expected stream false, got %v", req["stream"])
	}
}

func TestDeepSeekClientMalformedResponseReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`not json at all`))
	}))
	defer srv.Close()

	client := NewDeepSeekClient(DeepSeekConfig{
		APIKey:  "test-key",
		Model:   "deepseek-v4-flash",
		BaseURL: srv.URL,
	})

	results, err := client.Chat([]core.Message{{Role: "user", Content: "Hi"}})
	if err != nil {
		return
	}

	for r := range results {
		if r.Err != nil {
			return
		}
	}
	t.Fatal("expected an error from malformed response, got none")
}
