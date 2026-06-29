package tools_test

import (
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
	"github.com/DietKyle956/last-known-good/internal/tools"
)

type mockSandbox struct{}

func (m *mockSandbox) Start(string) (*sandbox.SessionHandle, error) {
	return nil, nil
}
func (m *mockSandbox) Exec(*sandbox.SessionHandle, string) (string, error) {
	return "", nil
}
func (m *mockSandbox) Stop(*sandbox.SessionHandle) error {
	return nil
}

func TestRegistryDispatchesToRegisteredTool(t *testing.T) {
	sb := &mockSandbox{}
	reg := tools.New(sb)
	reg.Register(tools.Tool{
		Name:        "echo",
		Description: "Echoes the input",
		Parameters:  map[string]any{},
		Execute: func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult {
			return core.ToolResult{
				ToolCallID: call.ID,
				Content:    call.Arguments,
			}
		},
	})

	result := reg.Execute(core.ToolCall{
		ID:        "call_1",
		Name:      "echo",
		Arguments: `"hello"`,
	})

	if result.ToolCallID != "call_1" {
		t.Fatalf("expected ToolCallID 'call_1', got %q", result.ToolCallID)
	}
	if result.Content != `"hello"` {
		t.Fatalf("expected content %q, got %q", `"hello"`, result.Content)
	}
	if result.IsError {
		t.Fatal("expected IsError to be false")
	}
}

func TestRegistryToolDefinitions(t *testing.T) {
	sb := &mockSandbox{}
	reg := tools.New(sb)
	reg.Register(tools.Tool{
		Name:        "read_file",
		Description: "Read a file from the workspace",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
		Execute: func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult {
			return core.ToolResult{}
		},
	})
	reg.Register(tools.Tool{
		Name:        "write_file",
		Description: "Write a file to the workspace",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
		},
		Execute: func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult {
			return core.ToolResult{}
		},
	})

	defs := reg.ToolDefinitions()
	if len(defs) != 2 {
		t.Fatalf("expected 2 definitions, got %d", len(defs))
	}

	var names []string
	for _, d := range defs {
		names = append(names, d.Name)
		if d.Description == "" {
			t.Errorf("missing description for tool %q", d.Name)
		}
		if d.Parameters == nil {
			t.Errorf("missing parameters for tool %q", d.Name)
		}
	}
	if len(names) != 2 || !contains(names, "read_file") || !contains(names, "write_file") {
		t.Fatalf("expected [read_file write_file], got %v", names)
	}
}

func TestRegistrySchemaValidationRejectsInvalidArgs(t *testing.T) {
	sb := &mockSandbox{}
	executed := false
	reg := tools.New(sb)
	reg.Register(tools.Tool{
		Name:        "read_file",
		Description: "Read a file",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []any{"path"},
		},
		Execute: func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult {
			executed = true
			return core.ToolResult{
				ToolCallID: call.ID,
				Content:    "ok",
			}
		},
	})

	result := reg.Execute(core.ToolCall{
		ID:        "call_1",
		Name:      "read_file",
		Arguments: `{}`,
	})
	if !result.IsError {
		t.Fatal("expected IsError for invalid args")
	}
	if executed {
		t.Fatal("tool was executed despite invalid arguments")
	}
}

func TestRegistrySchemaValidationAcceptsValidArgs(t *testing.T) {
	sb := &mockSandbox{}
	reg := tools.New(sb)
	reg.Register(tools.Tool{
		Name:        "read_file",
		Description: "Read a file",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []any{"path"},
		},
		Execute: func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult {
			return core.ToolResult{
				ToolCallID: call.ID,
				Content:    call.Arguments,
			}
		},
	})

	result := reg.Execute(core.ToolCall{
		ID:        "call_1",
		Name:      "read_file",
		Arguments: `{"path": "/foo"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
}

func TestRegistryIsReadOnlyDelegatesToTool(t *testing.T) {
	sb := &mockSandbox{}
	reg := tools.New(sb)
	reg.Register(tools.Tool{
		Name:       "ro_tool",
		IsReadOnly: true,
		Execute:    func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult { return core.ToolResult{} },
	})
	reg.Register(tools.Tool{
		Name:       "rw_tool",
		IsReadOnly: false,
		Execute:    func(_ sandbox.Sandbox, call core.ToolCall) core.ToolResult { return core.ToolResult{} },
	})

	if !reg.IsReadOnly("ro_tool") {
		t.Fatal("expected ro_tool to be read-only")
	}
	if reg.IsReadOnly("rw_tool") {
		t.Fatal("expected rw_tool to be writable")
	}
}

func TestRegistryUnknownToolReturnsError(t *testing.T) {
	sb := &mockSandbox{}
	reg := tools.New(sb)
	result := reg.Execute(core.ToolCall{
		ID:        "call_1",
		Name:      "nonexistent",
		Arguments: "{}",
	})
	if !result.IsError {
		t.Fatal("expected IsError to be true for unknown tool")
	}
	if result.Content != "unknown tool: nonexistent" {
		t.Fatalf("expected error about unknown tool, got %q", result.Content)
	}
}

func contains(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
