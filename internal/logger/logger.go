package logger

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/DietKyle956/last-known-good/internal/hooks"
)

type jsonLogEntry struct {
	Timestamp  string      `json:"timestamp"`
	SessionID  int64       `json:"session_id"`
	Type       string      `json:"type"`
	Model      string      `json:"model,omitempty"`
	ToolCall   *toolCall   `json:"tool_call,omitempty"`
	ToolResult *toolResult `json:"tool_result,omitempty"`
	Error      string      `json:"error,omitempty"`
}

type toolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type toolResult struct {
	ToolCallID string `json:"tool_call_id"`
	Content    string `json:"content"`
	IsError    bool   `json:"is_error"`
}

type Logger struct {
	mu        sync.Mutex
	f         *os.File
	encoder   *json.Encoder
	sessionID int64
}

func New(sessionID int64, dir string) (*Logger, error) {
	path := fmt.Sprintf("session_%d.jsonl", sessionID)
	if dir != "" {
		path = dir + "/" + path
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create log file: %w", err)
	}
	return &Logger{
		f:         f,
		encoder:   json.NewEncoder(f),
		sessionID: sessionID,
	}, nil
}

func (l *Logger) Close() error {
	return l.f.Close()
}

func (l *Logger) Hook(ev hooks.HookEvent) *hooks.HookResult {
	entry := jsonLogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		SessionID: l.sessionID,
		Type:      ev.Type.String(),
		Model:     ev.Model,
	}
	if ev.ToolCall != nil {
		entry.ToolCall = &toolCall{
			ID:        ev.ToolCall.ID,
			Name:      ev.ToolCall.Name,
			Arguments: ev.ToolCall.Arguments,
		}
	}
	if ev.ToolResult != nil {
		entry.ToolResult = &toolResult{
			ToolCallID: ev.ToolResult.ToolCallID,
			Content:    ev.ToolResult.Content,
			IsError:    ev.ToolResult.IsError,
		}
	}
	if ev.Error != nil {
		entry.Error = ev.Error.Error()
	}

	l.mu.Lock()
	err := l.encoder.Encode(entry)
	l.mu.Unlock()
	if err != nil {
		return nil
	}
	return nil
}
