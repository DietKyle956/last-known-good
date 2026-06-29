package llm

// DeepSeekRequest is the request body for the chat completions endpoint.
type DeepSeekRequest struct {
	Model           string            `json:"model"`
	Messages        []DeepSeekMessage `json:"messages"`
	Stream          bool              `json:"stream"`
	Thinking        *ThinkingConfig   `json:"thinking,omitempty"`
	ReasoningEffort string            `json:"reasoning_effort,omitempty"`
}

// ThinkingConfig controls thinking mode on a request.
type ThinkingConfig struct {
	Type string `json:"type"`
}

// DeepSeekMessage is a single message in the conversation.
type DeepSeekMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

// DeepSeekResponse is the response from a non-streaming chat completion.
type DeepSeekResponse struct {
	ID      string           `json:"id"`
	Choices []DeepSeekChoice `json:"choices"`
}

// DeepSeekChoice is a single choice in the response.
type DeepSeekChoice struct {
	Index        int             `json:"index"`
	Message      DeepSeekRespMsg `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

// DeepSeekRespMsg is the message inside a response choice.
type DeepSeekRespMsg struct {
	Role      string             `json:"role"`
	Content   string             `json:"content"`
	ToolCalls []DeepSeekToolCall `json:"tool_calls,omitempty"`
}

// DeepSeekToolCall is a tool call in a response message.
type DeepSeekToolCall struct {
	ID       string               `json:"id"`
	Type     string               `json:"type"`
	Function DeepSeekFunctionCall `json:"function"`
}

// DeepSeekFunctionCall is the function details in a tool call.
type DeepSeekFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// DeepSeekChunk is a streaming response chunk.
type DeepSeekChunk struct {
	ID      string        `json:"id"`
	Choices []ChunkChoice `json:"choices"`
}

// ChunkChoice is a single choice in a streaming chunk.
type ChunkChoice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason,omitempty"`
}

// Delta is the content delta in a streaming chunk.
type Delta struct {
	Content   string          `json:"content,omitempty"`
	ToolCalls []DeltaToolCall `json:"tool_calls,omitempty"`
}

// DeltaToolCall is a tool call delta in a streaming chunk.
type DeltaToolCall struct {
	Index    int               `json:"index"`
	ID       string            `json:"id,omitempty"`
	Type     string            `json:"type,omitempty"`
	Function DeltaFunctionCall `json:"function,omitempty"`
}

// DeltaFunctionCall is the function details in a tool call delta.
type DeltaFunctionCall struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}
