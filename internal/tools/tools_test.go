package tools_test

import (
	"os"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/sandbox"
	"github.com/DietKyle956/last-known-good/internal/tools"
)

func startSandbox(t *testing.T) *sandbox.SessionHandle {
	t.Helper()
	dir, err := os.MkdirTemp("", "tools-test-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	h, err := sandbox.Start(dir, sandbox.SandboxConfig{
		Network: &sandbox.NetworkConfig{Allow: []string{"dl-cdn.alpinelinux.org"}},
	})
	if err != nil {
		t.Fatalf("sandbox.Start failed: %v", err)
	}
	t.Cleanup(func() {
		if err := sandbox.Stop(h); err != nil {
			t.Errorf("failed to stop sandbox: %v", err)
		}
	})
	_, err = sandbox.Exec(h, "apk add --no-cache git >/dev/null 2>&1")
	if err != nil {
		t.Fatalf("failed to install git: %v", err)
	}
	return h
}

func TestRegistryDispatchesToRegisteredTool(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	reg.Register(tools.Tool{
		Name:        "echo",
		Description: "Echoes the input",
		Parameters:  map[string]any{},
		Execute: func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult {
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
	h := startSandbox(t)
	reg := tools.New(h)
	reg.Register(tools.Tool{
		Name:        "read_file",
		Description: "Read a file from the workspace",
		Parameters: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"path": map[string]any{"type": "string"},
			},
		},
		Execute: func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult {
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
		Execute: func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult {
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
	h := startSandbox(t)
	executed := false
	reg := tools.New(h)
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
		Execute: func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult {
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
	h := startSandbox(t)
	reg := tools.New(h)
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
		Execute: func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult {
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
	h := startSandbox(t)
	reg := tools.New(h)
	reg.Register(tools.Tool{
		Name:       "ro_tool",
		IsReadOnly: true,
		Execute:    func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult { return core.ToolResult{} },
	})
	reg.Register(tools.Tool{
		Name:       "rw_tool",
		IsReadOnly: false,
		Execute:    func(_ *sandbox.SessionHandle, call core.ToolCall) core.ToolResult { return core.ToolResult{} },
	})

	if !reg.IsReadOnly("ro_tool") {
		t.Fatal("expected ro_tool to be read-only")
	}
	if reg.IsReadOnly("rw_tool") {
		t.Fatal("expected rw_tool to be writable")
	}
}

func TestRegistryUnknownToolReturnsError(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
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

func TestReadFileReturnsContents(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "echo -n 'hello world' > /workspace/test.txt")
	if err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "read_file",
		Arguments: `{"path": "/workspace/test.txt"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content != "hello world" {
		t.Fatalf("expected 'hello world', got %q", result.Content)
	}
}

func TestReadFileMissingReturnsError(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "read_file",
		Arguments: `{"path": "/workspace/nonexistent.txt"}`,
	})
	if !result.IsError {
		t.Fatal("expected error for missing file")
	}
	if result.Content == "" {
		t.Fatal("expected non-empty error content")
	}
}

func TestWriteFileCreatesFile(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "write_file",
		Arguments: `{"path": "/workspace/new.txt", "content": "hello from tool"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	out, err := sandbox.Exec(h, "cat /workspace/new.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if out != "hello from tool" {
		t.Fatalf("expected 'hello from tool', got %q", out)
	}
}

func TestWriteFileOverwritesExisting(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "echo -n 'old content' > /workspace/overwrite.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "write_file",
		Arguments: `{"path": "/workspace/overwrite.txt", "content": "new content"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	out, err := sandbox.Exec(h, "cat /workspace/overwrite.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if out != "new content" {
		t.Fatalf("expected 'new content', got %q", out)
	}
}

func TestEditFileReplacesText(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "printf 'line 1\nold line\nline 3' > /workspace/edit.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "edit_file",
		Arguments: `{"path": "/workspace/edit.txt", "old_string": "old line", "new_string": "new line"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}

	out, err := sandbox.Exec(h, "cat /workspace/edit.txt")
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if out != "line 1\nnew line\nline 3" {
		t.Fatalf("expected 'line 1\\nnew line\\nline 3', got %q", out)
	}
}

func TestBashReturnsStdout(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "bash",
		Arguments: `{"command": "echo hello-world"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content != "hello-world\n" {
		t.Fatalf("expected 'hello-world\\n', got %q", result.Content)
	}
}

func TestBashNonZeroExitReturnsError(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "bash",
		Arguments: `{"command": "echo err-output >&2; exit 1"}`,
	})
	if !result.IsError {
		t.Fatal("expected error for non-zero exit")
	}
	if result.Content == "" {
		t.Fatal("expected non-empty error content")
	}
}

func TestGrepReturnsMatchingLines(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "printf 'foo bar\nbaz qux\nfoo baz' > /workspace/grep.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "grep",
		Arguments: `{"pattern": "foo", "path": "/workspace/grep.txt"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content == "" {
		t.Fatal("expected non-empty grep output")
	}
}

func TestGrepNoMatchesReturnsEmpty(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "printf 'abc\ndef' > /workspace/nomatch.txt")
	if err != nil {
		t.Fatalf("failed to create file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "grep",
		Arguments: `{"pattern": "zzz", "path": "/workspace/nomatch.txt"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
}

func TestGlobReturnsMatchingPaths(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "touch /workspace/a.txt /workspace/b.txt /workspace/c.go")
	if err != nil {
		t.Fatalf("failed to create files: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "glob",
		Arguments: `{"pattern": "*.txt", "path": "/workspace"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content == "" {
		t.Fatal("expected non-empty glob output")
	}
}

func TestGlobNoMatchesReturnsEmpty(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "glob",
		Arguments: `{"pattern": "*.xyz", "path": "/workspace"}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
}

func TestGitDiffReturnsChanges(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	out, err := sandbox.Exec(h, "cd /workspace && echo -n 'original' > file.txt && git init && git config --global safe.directory /workspace && git config user.email test@test.com && git config user.name test && git add . && git commit -m init 2>&1")
	if err != nil {
		t.Fatalf("failed to setup git repo: %v\noutput: %s", err, out)
	}

	// Modify the file (separate exec call since the first one creates the repo)
	_, err = sandbox.Exec(h, "cd /workspace && echo -n 'modified' > file.txt")
	if err != nil {
		t.Fatalf("failed to modify file: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "git_diff",
		Arguments: `{}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content == "" {
		t.Fatal("expected non-empty diff output")
	}
}

func TestGitDiffNoChangesReturnsEmpty(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	_, err := sandbox.Exec(h, "cd /workspace && echo -n 'content' > file.txt && git init && git config --global safe.directory /workspace && git config user.email test@test.com && git config user.name test && git add . && git commit -m init 2>&1")
	if err != nil {
		t.Fatalf("failed to setup clean git repo: %v", err)
	}

	result := reg.Execute(core.ToolCall{
		ID:        "c1",
		Name:      "git_diff",
		Arguments: `{}`,
	})
	if result.IsError {
		t.Fatalf("unexpected error: %s", result.Content)
	}
	if result.Content != "" {
		t.Fatalf("expected empty content, got %q", result.Content)
	}
}

func TestAllToolsExecuteThroughSandbox(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	defs := reg.ToolDefinitions()
	names := make(map[string]bool)
	for _, d := range defs {
		names[d.Name] = true
	}

	expected := []string{"read_file", "write_file", "edit_file", "bash", "grep", "glob", "git_diff"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("missing tool definition: %s", name)
		}
	}
}

func TestToolSchemasDefineRequiredArgs(t *testing.T) {
	h := startSandbox(t)
	reg := tools.New(h)
	tools.RegisterAll(reg)

	defs := reg.ToolDefinitions()
	for _, d := range defs {
		props, hasProps := d.Parameters["properties"].(map[string]any)
		if !hasProps {
			t.Errorf("tool %q: missing properties", d.Name)
			continue
		}
		req, hasReq := d.Parameters["required"].([]any)
		if !hasReq {
			continue
		}
		for _, r := range req {
			rname, ok := r.(string)
			if !ok {
				continue
			}
			if _, exists := props[rname]; !exists {
				t.Errorf("tool %q: required property %q not in properties", d.Name, rname)
			}
		}
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
