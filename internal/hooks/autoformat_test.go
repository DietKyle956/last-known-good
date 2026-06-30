package hooks

import (
	"context"
	"errors"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
)

type mockFormatterExecer struct {
	fn func(ctx context.Context, command string) (string, error)
}

func (m *mockFormatterExecer) Exec(ctx context.Context, command string) (string, error) {
	return m.fn(ctx, command)
}

func TestAutoFormatHookRunsFormatterForGoFile(t *testing.T) {
	var executedCommand string
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executedCommand = cmd
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"main.go","content":"package main\nfunc main() {}"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if r != nil {
		t.Fatalf("expected nil result from AfterToolCall, got %v", r)
	}
	if executedCommand != "gofmt -w 'main.go'" {
		t.Fatalf("expected gofmt on main.go, got: %q", executedCommand)
	}
}

func TestAutoFormatHookSkipsUnrecognizedExtension(t *testing.T) {
	var executed bool
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executed = true
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"notes.txt","content":"hello"}`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if executed {
		t.Fatal("expected no formatter to run for .txt file")
	}
}

func TestAutoFormatHookSkipsNonWriteFileTool(t *testing.T) {
	var executed bool
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executed = true
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"ls"}`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if executed {
		t.Fatal("expected no formatter to run for bash tool call")
	}
}

func TestAutoFormatHookSkipsNilToolCall(t *testing.T) {
	var executed bool
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executed = true
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: nil})
	if executed {
		t.Fatal("expected no formatter to run for nil tool call")
	}
}

func TestAutoFormatHookSkipsInvalidJSON(t *testing.T) {
	var executed bool
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executed = true
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `not-json`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if executed {
		t.Fatal("expected no formatter to run for invalid JSON")
	}
}

func TestAutoFormatHookCallsOnFailureWhenFormatterFails(t *testing.T) {
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		return "error output", errors.New("formatter failed")
	}}

	var failurePath, failureCommand string
	h := NewAutoFormatHook(ex, nil, func(path string, command string, err error) {
		failurePath = path
		failureCommand = command
	})
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"main.go","content":"package main"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if r != nil {
		t.Fatalf("expected nil result from AfterToolCall even on failure, got %v", r)
	}
	if failurePath != "main.go" {
		t.Fatalf("expected failure path main.go, got %q", failurePath)
	}
	if failureCommand != "gofmt -w 'main.go'" {
		t.Fatalf("expected failure command gofmt, got %q", failureCommand)
	}
}

func TestAutoFormatHookDoesNotCrashOnFormatterFailure(t *testing.T) {
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		return "out of memory", errors.New("formatter crashed")
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"main.go","content":"package main"}`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
}

func TestAutoFormatHookCustomFormatters(t *testing.T) {
	var executedCommand string
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executedCommand = cmd
		return "", nil
	}}

	formatters := map[string]string{
		".py": "black -q %s",
		".js": "prettier --write %s",
	}
	h := NewAutoFormatHook(ex, formatters, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"app.py","content":"x=1"}`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if executedCommand != "black -q 'app.py'" {
		t.Fatalf("expected black on app.py, got: %q", executedCommand)
	}
}

func TestAutoFormatHookCustomFormattersUnrecognized(t *testing.T) {
	var executed bool
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		executed = true
		return "", nil
	}}

	formatters := map[string]string{
		".py": "black -q %s",
	}
	h := NewAutoFormatHook(ex, formatters, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"app.js","content":"x=1"}`,
	}
	s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if executed {
		t.Fatal("expected no formatter to run for .js with only .py configured")
	}
}

func TestAutoFormatHookRegisteredViaSystem(t *testing.T) {
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		return "", nil
	}}

	h := NewAutoFormatHook(ex, nil, nil)
	s := New(nil)
	s.Register(AfterToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "write_file",
		Arguments: `{"path":"test.go","content":"package test"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: AfterToolCall, ToolCall: tc})
	if r != nil {
		t.Fatal("expected nil result from AfterToolCall")
	}
}

func TestDefaultFormatters(t *testing.T) {
	fmts := DefaultFormatters()
	if fmts == nil {
		t.Fatal("expected non-nil default formatters")
	}
	cmd, ok := fmts[".go"]
	if !ok {
		t.Fatal("expected default formatters to include .go")
	}
	if cmd != "gofmt -w %s" {
		t.Fatalf("expected gofmt command, got %q", cmd)
	}
}

func TestFormattersReturnsCopy(t *testing.T) {
	ex := &mockFormatterExecer{fn: func(_ context.Context, cmd string) (string, error) {
		return "", nil
	}}
	h := NewAutoFormatHook(ex, nil, nil)
	fmts := h.Formatters()
	fmts["new"] = "value"
	original := h.Formatters()
	if _, ok := original["new"]; ok {
		t.Fatal("Formatters() should return a copy, not the original map")
	}
}
