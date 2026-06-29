package llm

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/DietKyle956/last-known-good/internal/core"
)

// DeepSeekConfig configures a DeepSeek client.
type DeepSeekConfig struct {
	APIKey          string
	Model           string
	BaseURL         string
	Stream          bool
	ThinkingMode    bool
	ReasoningEffort string
}

// DeepSeekClient implements core.LLM for DeepSeek's API.
type DeepSeekClient struct {
	config DeepSeekConfig
	http   *http.Client
}

// NewDeepSeekClient creates a new DeepSeek client.
func NewDeepSeekClient(config DeepSeekConfig) *DeepSeekClient {
	if config.BaseURL == "" {
		config.BaseURL = "https://api.deepseek.com/v1"
	}
	return &DeepSeekClient{
		config: config,
		http:   &http.Client{},
	}
}

// Chat sends a chat completion request to DeepSeek and returns a result channel.
func (c *DeepSeekClient) Chat(messages []core.Message) (<-chan core.Result, error) {
	req, err := c.buildRequest(messages)
	if err != nil {
		return nil, err
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("deepseek request failed: %w", err)
	}

	results := make(chan core.Result, 64)

	if c.config.Stream {
		go c.readStream(resp, results)
	} else {
		go c.readResponse(resp, results)
	}

	return results, nil
}

func (c *DeepSeekClient) buildRequest(messages []core.Message) (*http.Request, error) {
	dsReq := DeepSeekRequest{
		Model:           c.config.Model,
		Messages:        make([]DeepSeekMessage, 0, len(messages)),
		Stream:          c.config.Stream,
		ReasoningEffort: c.config.ReasoningEffort,
	}

	if c.config.ThinkingMode {
		dsReq.Thinking = &ThinkingConfig{Type: "enabled"}
	}

	for _, m := range messages {
		dsm := DeepSeekMessage{
			Role:    m.Role,
			Content: m.Content,
		}
		if m.ToolResult != nil {
			dsm.ToolCallID = m.ToolResult.ToolCallID
			if dsm.Content == "" {
				dsm.Content = m.ToolResult.Content
			}
		}
		dsReq.Messages = append(dsReq.Messages, dsm)
	}

	body, err := json.Marshal(dsReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", c.config.BaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	return req, nil
}

func (c *DeepSeekClient) readResponse(resp *http.Response, results chan<- core.Result) {
	defer resp.Body.Close()
	defer close(results)

	var dsResp DeepSeekResponse
	if err := json.NewDecoder(resp.Body).Decode(&dsResp); err != nil {
		results <- core.Result{Err: fmt.Errorf("decode response: %w", err)}
		return
	}

	if len(dsResp.Choices) == 0 {
		results <- core.Result{Done: true}
		return
	}

	choice := dsResp.Choices[0]
	if choice.Message.Content != "" {
		results <- core.Result{Content: choice.Message.Content, IsChunk: true}
	}

	toolCalls := convertToolCalls(choice.Message.ToolCalls)
	results <- core.Result{Done: true, ToolCalls: toolCalls}
}

func (c *DeepSeekClient) readStream(resp *http.Response, results chan<- core.Result) {
	defer resp.Body.Close()
	defer close(results)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			results <- core.Result{Done: true}
			return
		}

		var chunk DeepSeekChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			results <- core.Result{Err: fmt.Errorf("decode chunk: %w", err)}
			return
		}

		if len(chunk.Choices) == 0 {
			continue
		}

		delta := chunk.Choices[0].Delta
		if delta.Content != "" {
			results <- core.Result{Content: delta.Content, IsChunk: true}
		}
	}

	if err := scanner.Err(); err != nil {
		results <- core.Result{Err: fmt.Errorf("stream read error: %w", err)}
	}
}

func convertToolCalls(tcs []DeepSeekToolCall) []core.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	calls := make([]core.ToolCall, 0, len(tcs))
	for _, tc := range tcs {
		calls = append(calls, core.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return calls
}
