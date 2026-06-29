package tools

import (
	"encoding/json"
	"fmt"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
)

var _ agent.ToolExecutor = (*Registry)(nil)

type ToolFn func(sandbox.Sandbox, core.ToolCall) core.ToolResult

type Tool struct {
	Name        string
	Description string
	Parameters  map[string]any
	IsReadOnly  bool
	Execute     ToolFn
}

type Registry struct {
	tools   map[string]Tool
	sandbox sandbox.Sandbox
}

func New(sb sandbox.Sandbox) *Registry {
	return &Registry{
		tools:   make(map[string]Tool),
		sandbox: sb,
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
	return defs
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name] = t
}

func (r *Registry) IsReadOnly(name string) bool {
	tool, ok := r.tools[name]
	return ok && tool.IsReadOnly
}

func (r *Registry) Execute(call core.ToolCall) core.ToolResult {
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
	return tool.Execute(r.sandbox, call)
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
