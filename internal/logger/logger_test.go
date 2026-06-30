package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/hooks"
)

func TestLogger_NewCreatesFile(t *testing.T) {
	dir := t.TempDir()
	l, err := New(42, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	info, err := os.Stat(filepath.Join(dir, "session_42.jsonl"))
	if err != nil {
		t.Fatal("expected log file to exist:", err)
	}
	if info.Size() != 0 {
		t.Fatal("expected empty log file")
	}
}

func TestLogger_FileReadableWhileActive(t *testing.T) {
	dir := t.TempDir()
	l, err := New(2, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Hook(hooks.HookEvent{Type: hooks.SessionStarted})
	l.Hook(hooks.HookEvent{Type: hooks.BeforeModelCall})

	data, err := os.ReadFile(filepath.Join(dir, "session_2.jsonl"))
	if err != nil {
		t.Fatal("should be able to read file while logger is active:", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
}

func TestLogger_WritesJSONL(t *testing.T) {
	dir := t.TempDir()
	l, err := New(1, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Hook(hooks.HookEvent{Type: hooks.SessionStarted})
	l.Close()

	data, err := os.ReadFile(filepath.Join(dir, "session_1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal("invalid JSON:", err)
	}
	if entry["session_id"].(float64) != 1 {
		t.Fatal("wrong session_id")
	}
	if entry["type"] != "session_started" {
		t.Fatal("wrong type")
	}
	if _, ok := entry["timestamp"]; !ok {
		t.Fatal("expected timestamp field")
	}
}

func TestLogger_SeparateFilesPerSession(t *testing.T) {
	dir := t.TempDir()

	l1, err := New(10, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l1.Close()

	l2, err := New(20, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l2.Close()

	l1.Hook(hooks.HookEvent{Type: hooks.SessionStarted})
	l2.Hook(hooks.HookEvent{Type: hooks.SessionStarted})

	for _, id := range []int64{10, 20} {
		path := filepath.Join(dir, fmt.Sprintf("session_%d.jsonl", id))
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("expected file session_%d.jsonl to exist: %v", id, err)
		}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("session_%d.jsonl: expected 1 line, got %d", id, len(lines))
		}
	}
}

func TestLogger_AllHookTypesLogged(t *testing.T) {
	dir := t.TempDir()
	l, err := New(3, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	types := []hooks.HookType{
		hooks.SessionStarted,
		hooks.SessionEnded,
		hooks.BeforeModelCall,
		hooks.AfterModelCall,
		hooks.BeforeToolCall,
		hooks.AfterToolCall,
	}
	for _, tt := range types {
		l.Hook(hooks.HookEvent{Type: tt})
	}
	l.Close()

	data, err := os.ReadFile(filepath.Join(dir, "session_3.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(types) {
		t.Fatalf("expected %d lines, got %d", len(types), len(lines))
	}
	for i, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("line %d: invalid JSON: %v", i, err)
		}
		if entry["type"] != types[i].String() {
			t.Fatalf("line %d: expected type %q, got %q", i, types[i].String(), entry["type"])
		}
		if entry["session_id"].(float64) != 3 {
			t.Fatalf("line %d: wrong session_id", i)
		}
	}
}

func TestLogger_ToolCallFieldsLogged(t *testing.T) {
	dir := t.TempDir()
	l, err := New(4, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	tc := &core.ToolCall{ID: "call_1", Name: "read_file", Arguments: `{"path":"test.txt"}`}
	l.Hook(hooks.HookEvent{Type: hooks.BeforeToolCall, ToolCall: tc})

	tr := &core.ToolResult{ToolCallID: "call_1", Content: "file contents", IsError: false}
	l.Hook(hooks.HookEvent{Type: hooks.AfterToolCall, ToolCall: tc, ToolResult: tr})
	l.Close()

	data, err := os.ReadFile(filepath.Join(dir, "session_4.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}

	var before, after map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(lines[1]), &after); err != nil {
		t.Fatal(err)
	}

	if before["type"] != "before_tool_call" {
		t.Fatal("expected first line to be before_tool_call")
	}
	tcField := before["tool_call"].(map[string]any)
	if tcField["id"] != "call_1" || tcField["name"] != "read_file" || tcField["arguments"] != `{"path":"test.txt"}` {
		t.Fatal("tool_call fields mismatch in before event")
	}

	if after["type"] != "after_tool_call" {
		t.Fatal("expected second line to be after_tool_call")
	}
	trField := after["tool_result"].(map[string]any)
	if trField["tool_call_id"] != "call_1" || trField["content"] != "file contents" || trField["is_error"] != false {
		t.Fatal("tool_result fields mismatch in after event")
	}
}

func TestLogger_ErrorFieldLogged(t *testing.T) {
	dir := t.TempDir()
	l, err := New(5, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Hook(hooks.HookEvent{Type: hooks.AfterModelCall, Error: os.ErrNotExist})
	l.Close()

	data, err := os.ReadFile(filepath.Join(dir, "session_5.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["error"] != "file does not exist" {
		t.Fatalf("expected error field %q, got %q", "file does not exist", entry["error"])
	}
}

func TestLogger_ModelFieldLogged(t *testing.T) {
	dir := t.TempDir()
	l, err := New(6, dir)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Hook(hooks.HookEvent{Type: hooks.BeforeModelCall, Model: "deepseek-v4-flash"})
	l.Close()

	data, err := os.ReadFile(filepath.Join(dir, "session_6.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}

	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	if entry["model"] != "deepseek-v4-flash" {
		t.Fatalf("expected model %q, got %q", "deepseek-v4-flash", entry["model"])
	}
}

func TestLogger_FilePersistsAfterClose(t *testing.T) {
	dir := t.TempDir()

	func() {
		l, err := New(7, dir)
		if err != nil {
			t.Fatal(err)
		}
		l.Hook(hooks.HookEvent{Type: hooks.SessionStarted})
		l.Close()
	}()

	_, err := os.Stat(filepath.Join(dir, "session_7.jsonl"))
	if err != nil {
		t.Fatal("log file should persist after logger is closed:", err)
	}
}
