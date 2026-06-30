package hooks

import (
	"context"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
)

func TestDangerousCommandHookBlocksRmRFSlash(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"rm -rf /"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r == nil {
		t.Fatal("expected dangerous command to be blocked, but it was not")
	}
	if !r.Block {
		t.Fatal("expected Block=true")
	}
	if r.Reason == "" {
		t.Fatal("expected a non-empty reason explaining why the command was blocked")
	}
}

func TestDangerousCommandHookBlocksRmRFTilde(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"rm -rf ~/important"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r == nil {
		t.Fatal("expected dangerous command to be blocked")
	}
}

func TestDangerousCommandHookBlocksForkBomb(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":":(){ :|:& };:"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r == nil {
		t.Fatal("expected fork bomb command to be blocked")
	}
}

func TestDangerousCommandHookBlocksMkfs(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"mkfs.ext4 /dev/sda1"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r == nil {
		t.Fatal("expected mkfs command to be blocked")
	}
}

func TestDangerousCommandHookAllowsSafeCommand(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"ls -la"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r != nil {
		t.Fatalf("expected safe command not to be blocked, got block with reason: %q", r.Reason)
	}
}

func TestDangerousCommandHookAllowsNonBashTool(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "read_file",
		Arguments: `{"path":"test.txt"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r != nil {
		t.Fatalf("expected non-bash tool call not to be blocked, got block with reason: %q", r.Reason)
	}
}

func TestDangerousCommandHookCustomPatterns(t *testing.T) {
	patterns := []string{`custom-dangerous`}
	h := NewDangerousCommandHook(patterns)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"echo custom-dangerous-command"}`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r == nil {
		t.Fatal("expected custom dangerous command to be blocked")
	}

	// A safe command without the custom pattern should pass
	tc2 := &core.ToolCall{
		Name:      "bash",
		Arguments: `{"command":"echo safe"}`,
	}
	r2 := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc2})
	if r2 != nil {
		t.Fatalf("expected safe command with custom patterns not to be blocked, got: %q", r2.Reason)
	}
}

func TestDangerousCommandHookHandlesNilToolCall(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: nil})
	if r != nil {
		t.Fatal("expected nil tool call not to cause a block")
	}
}

func TestDangerousCommandHookHandlesInvalidJSON(t *testing.T) {
	h := NewDangerousCommandHook(nil)
	s := New(nil)
	s.Register(BeforeToolCall, h.Handler)

	tc := &core.ToolCall{
		Name:      "bash",
		Arguments: `not-json`,
	}
	r := s.Notify(context.Background(), HookEvent{Type: BeforeToolCall, ToolCall: tc})
	if r != nil {
		t.Fatalf("expected invalid JSON not to be blocked, got: %q", r.Reason)
	}
}
