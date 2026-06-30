package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/DietKyle956/last-known-good/internal/sandbox"
)

// AutoFormatHook is an AfterToolCall hook that runs a language formatter
// inside the sandbox after a file write to a recognized source file type.
type AutoFormatHook struct {
	execer     sandbox.Execer
	formatters map[string]string // ext -> formatter command template (e.g. ".go" -> "gofmt -w %s")
	onFailure  func(path string, command string, err error)
}

// NewAutoFormatHook creates an auto-format hook with the given execer and
// extension-to-formatter mapping. If formatters is nil, DefaultFormatters is used.
// When a formatter fails, onFailure is called if non-nil.
func NewAutoFormatHook(execer sandbox.Execer, formatters map[string]string, onFailure func(path string, command string, err error)) *AutoFormatHook {
	if formatters == nil {
		formatters = DefaultFormatters()
	}
	return &AutoFormatHook{
		execer:     execer,
		formatters: formatters,
		onFailure:  onFailure,
	}
}

// DefaultFormatters returns the default set of extension-to-formatter mappings.
func DefaultFormatters() map[string]string {
	return map[string]string{
		".go": "gofmt -w %s",
	}
}

// Handler returns an AfterToolCall hook function that auto-formats written files.
func (h *AutoFormatHook) Handler(event HookEvent) *HookResult {
	if event.ToolCall == nil || event.ToolCall.Name != "write_file" {
		return nil
	}

	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(event.ToolCall.Arguments), &args); err != nil {
		return nil
	}

	ext := filepath.Ext(args.Path)
	cmdTemplate, ok := h.formatters[ext]
	if !ok {
		return nil
	}

	cmd := fmt.Sprintf(cmdTemplate, shellQuote(args.Path))
	_, err := h.execer.Exec(context.Background(), cmd)
	if err != nil {
		if h.onFailure != nil {
			h.onFailure(args.Path, cmd, err)
		}
	}

	return nil
}

// Formatters returns a copy of the configured formatters map.
func (h *AutoFormatHook) Formatters() map[string]string {
	result := make(map[string]string, len(h.formatters))
	for k, v := range h.formatters {
		result[k] = v
	}
	return result
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\\''") + "'"
}
