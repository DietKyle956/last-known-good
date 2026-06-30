package hooks

import (
	"encoding/json"
	"fmt"
	"regexp"
)

// DangerousCommandHook is a BeforeToolCall hook that blocks shell commands
// matching known dangerous patterns.
type DangerousCommandHook struct {
	patterns []string
	compiled []*regexp.Regexp
}

// NewDangerousCommandHook creates a hook with the given dangerous patterns.
// If patterns is nil, DefaultDangerousPatterns are used.
func NewDangerousCommandHook(patterns []string) *DangerousCommandHook {
	if patterns == nil {
		patterns = DefaultDangerousPatterns()
	}
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return &DangerousCommandHook{patterns: patterns, compiled: compiled}
}

// DefaultDangerousPatterns returns the default set of dangerous command patterns.
func DefaultDangerousPatterns() []string {
	return []string{
		`rm\s+-rf\s+[/~]`,
		`rm\s+-rf\s+/\*`,
		`mkfs\.\w+`,
		`dd\s+if=.*\s+of=`,
		`:\(\)\s*\{[^}]*\};?\s*:`,
		`chmod\s+-R\s+777\s+/`,
		`wget\s+.*\|\s*(bash|sh)`,
		`curl\s+.*\|\s*(bash|sh)`,
		`>\s*/dev/(sda|sdb|sdc|sdd|nvme)`,
	}
}

// Handler returns a BeforeToolCall hook function that checks bash commands
// against the configured dangerous patterns.
func (h *DangerousCommandHook) Handler(event HookEvent) *HookResult {
	if event.ToolCall == nil || event.ToolCall.Name != "bash" {
		return nil
	}

	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(event.ToolCall.Arguments), &args); err != nil {
		return nil
	}

	for i, re := range h.compiled {
		if re.MatchString(args.Command) {
			return &HookResult{
				Block:  true,
				Reason: fmt.Sprintf("blocked: command matches dangerous pattern %q", h.patterns[i]),
			}
		}
	}
	return nil
}

// Patterns returns the configured patterns (for inspection).
func (h *DangerousCommandHook) Patterns() []string {
	result := make([]string, len(h.patterns))
	copy(result, h.patterns)
	return result
}
