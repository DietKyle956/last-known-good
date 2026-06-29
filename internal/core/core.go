package core

// Message represents a single message in the conversation.
type Message struct {
	Role       string
	Content    string
	ToolCalls  []ToolCall
	ToolResult *ToolResult
}

// ToolCall represents a tool call requested by the model.
type ToolCall struct {
	ID        string
	Name      string
	Arguments string
}

// ToolResult represents the result of a tool execution.
type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
	Metadata   map[string]any
}

// Result is a single item from the LLM stream.
type Result struct {
	Content   string
	ToolCalls []ToolCall
	IsChunk   bool
	Done      bool
	Err       error
}
