package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
)

var _ agent.ToolExecutor = (*Registry)(nil)

type ToolFn func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	IsReadOnly  bool
	Execute     ToolFn
}

type Registry struct {
	tools map[string]Tool
	shell sandbox.Execer
}

func New(shell sandbox.Execer) *Registry {
	return &Registry{
		tools: make(map[string]Tool),
		shell: shell,
	}
}

type ToolDefinition struct {
	Name        string
	Description string
	Parameters  map[string]any
}

func (r *Registry) ToolDefinitions() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, t := range r.tools {
		defs = append(defs, ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		})
	}
	sort.Slice(defs, func(i, j int) bool {
		return defs[i].Name < defs[j].Name
	})
	return defs
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name] = t
}

func (r *Registry) Restrict(allowed []string) {
	filtered := make(map[string]Tool, len(allowed))
	for _, name := range allowed {
		if t, ok := r.tools[name]; ok {
			filtered[name] = t
		}
	}
	r.tools = filtered
}

func (r *Registry) IsReadOnly(name string) bool {
	tool, ok := r.tools[name]
	return ok && tool.IsReadOnly
}

func (r *Registry) Execute(ctx context.Context, call core.ToolCall) core.ToolResult {
	tool, ok := r.tools[call.Name]
	if !ok {
		return core.ToolResult{
			ToolCallID: call.ID,
			IsError:    true,
			Content:    "unknown tool: " + call.Name,
		}
	}
	if err := validateArgs(tool.Parameters, call.Arguments); err != nil {
		return core.ToolResult{
			ToolCallID: call.ID,
			IsError:    true,
			Content:    err.Error(),
		}
	}
	return tool.Execute(ctx, r.shell, call)
}

func validateArgs(schema map[string]any, raw string) error {
	var args any
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if schema == nil {
		return nil
	}

	if t, _ := schema["type"].(string); t == "object" {
		obj, ok := args.(map[string]any)
		if !ok {
			return fmt.Errorf("expected object, got %T", args)
		}
		if req, ok := schema["required"].([]any); ok {
			for _, r := range req {
				name, _ := r.(string)
				if _, exists := obj[name]; !exists {
					return fmt.Errorf("missing required property: %q", name)
				}
			}
		}
		if props, ok := schema["properties"].(map[string]any); ok {
			for key, val := range obj {
				if propSchema, ok := props[key].(map[string]any); ok {
					if err := validateProperty(propSchema, key, val); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

func validateProperty(schema map[string]any, name string, val any) error {
	expectedType, _ := schema["type"].(string)
	if expectedType == "" {
		return nil
	}
	switch expectedType {
	case "string":
		if _, ok := val.(string); !ok {
			return fmt.Errorf("property %q: expected string, got %T", name, val)
		}
	case "number":
		switch val.(type) {
		case float64, int, int64:
		default:
			return fmt.Errorf("property %q: expected number, got %T", name, val)
		}
	case "boolean":
		if _, ok := val.(bool); !ok {
			return fmt.Errorf("property %q: expected boolean, got %T", name, val)
		}
	case "array":
		if _, ok := val.([]any); !ok {
			return fmt.Errorf("property %q: expected array, got %T", name, val)
		}
	case "object":
		if _, ok := val.(map[string]any); !ok {
			return fmt.Errorf("property %q: expected object, got %T", name, val)
		}
	}
	return nil
}

func RegisterAll(r *Registry) {
	r.Register(readFileTool())
	r.Register(writeFileTool())
	r.Register(editFileTool())
	r.Register(bashTool())
	r.Register(grepTool())
	r.Register(globTool())
	r.Register(gitDiffTool())
}

func readFileTool() Tool {
	return Tool{
		Name:        "read_file",
		Description: "Read a file from the workspace inside the sandbox",
		IsReadOnly:  true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
			"required": []any{"path"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			out, err := ex.Exec(ctx, "cat "+quote(args.Path))
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			return core.ToolResult{ToolCallID: call.ID, Content: out}
		},
	}
}

func writeFileTool() Tool {
	return Tool{
		Name:        "write_file",
		Description: "Write a file to the workspace inside the sandbox",
		IsReadOnly:  false,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":    map[string]any{"type": "string"},
				"content": map[string]any{"type": "string"},
			},
			"required": []any{"path", "content"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			cmd := "printf '%s' " + quote(args.Content) + " > " + quote(args.Path)
			out, err := ex.Exec(ctx, cmd)
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			return core.ToolResult{ToolCallID: call.ID, Content: "written"}
		},
	}
}

func editFileTool() Tool {
	return Tool{
		Name:        "edit_file",
		Description: "Edit a file in the workspace by finding and replacing text",
		IsReadOnly:  false,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path":       map[string]any{"type": "string"},
				"old_string": map[string]any{"type": "string"},
				"new_string": map[string]any{"type": "string"},
			},
			"required": []any{"path", "old_string", "new_string"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Path     string `json:"path"`
				OldValue string `json:"old_string"`
				NewValue string `json:"new_string"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			cmd := "sed -i 's/" + escapeSed(args.OldValue) + "/" + escapeSed(args.NewValue) + "/g' " + quote(args.Path)
			out, err := ex.Exec(ctx, cmd)
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			return core.ToolResult{ToolCallID: call.ID, Content: "edited"}
		},
	}
}

func bashTool() Tool {
	return Tool{
		Name:        "bash",
		Description: "Run a shell command inside the sandbox",
		IsReadOnly:  false,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"command": map[string]any{"type": "string"},
			},
			"required": []any{"command"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Command string `json:"command"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			out, err := ex.Exec(ctx, args.Command+" 2>&1")
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			return core.ToolResult{ToolCallID: call.ID, Content: out}
		},
	}
}

func grepTool() Tool {
	return Tool{
		Name:        "grep",
		Description: "Search for a pattern in a file inside the sandbox",
		IsReadOnly:  true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
				"path":    map[string]any{"type": "string"},
			},
			"required": []any{"pattern", "path"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			cmd := "grep -rn " + quote(args.Pattern) + " " + quote(args.Path) + " 2>/dev/null || true"
			out, err := ex.Exec(ctx, cmd)
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			out = strings.TrimSuffix(out, "\n")
			return core.ToolResult{ToolCallID: call.ID, Content: out}
		},
	}
}

func globTool() Tool {
	return Tool{
		Name:        "glob",
		Description: "List files matching a glob pattern inside the sandbox",
		IsReadOnly:  true,
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"pattern": map[string]any{"type": "string"},
				"path":    map[string]any{"type": "string"},
			},
			"required": []any{"pattern", "path"},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			var args struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal([]byte(call.Arguments), &args); err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: err.Error()}
			}
			cmd := "find " + quote(args.Path) + " -type f -name " + quote(args.Pattern) + " 2>/dev/null || true"
			out, err := ex.Exec(ctx, cmd)
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			out = strings.TrimSuffix(out, "\n")
			return core.ToolResult{ToolCallID: call.ID, Content: out}
		},
	}
}

func gitDiffTool() Tool {
	return Tool{
		Name:        "git_diff",
		Description: "Show unstaged changes in the git repository inside the sandbox",
		IsReadOnly:  true,
		Parameters: map[string]any{
			"type":       "object",
			"properties": map[string]any{},
		},
		Execute: func(ctx context.Context, ex sandbox.Execer, call core.ToolCall) core.ToolResult {
			out, err := ex.Exec(ctx, "cd /workspace && git diff 2>/dev/null || true")
			if err != nil {
				return core.ToolResult{ToolCallID: call.ID, IsError: true, Content: out}
			}
			out = strings.TrimSuffix(out, "\n")
			return core.ToolResult{ToolCallID: call.ID, Content: out}
		},
	}
}

func quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}

func escapeSed(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "/", "\\/")
	s = strings.ReplaceAll(s, "&", "\\&")
	return s
}
