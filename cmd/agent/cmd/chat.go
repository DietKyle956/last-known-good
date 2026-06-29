package cmd

import (
	"fmt"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/DietKyle956/last-known-good/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var modelFlag string

type stubExecutor struct{}

func (s *stubExecutor) Execute(call core.ToolCall) core.ToolResult {
	return core.ToolResult{
		ToolCallID: call.ID,
		IsError:    true,
		Content:    "tool execution not available in chat mode",
	}
}

func (s *stubExecutor) IsReadOnly(name string) bool {
	return false
}

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive chat session",
	Long:  "Start an interactive terminal UI session with the agent.",
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := llm.APIKeyForModel(modelFlag)
		if apiKey == "" {
			return fmt.Errorf("no API key found for model %q — set DEEPSEEK_FLASH_KEY, DEEPSEEK_PRO_KEY, or DEEPSEEK_API_KEY", modelFlag)
		}

		client := llm.NewDeepSeekClient(llm.DeepSeekConfig{
			APIKey: apiKey,
			Model:  modelFlag,
			Stream: true,
		})

		messages := []core.Message{
			{Role: "system", Content: "You are a helpful assistant."},
		}

		return runTUI(client, messages)
	},
}

func runTUI(llmClient agent.LLM, messages []core.Message) error {
	submit := make(chan string, 64)
	events := make(chan agent.AgentEvent, 128)
	exec := &stubExecutor{}

	go coordinator(llmClient, exec, &messages, events, submit)

	m := tui.New(events, submit)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func coordinator(llmClient agent.LLM, exec agent.ToolExecutor, messages *[]core.Message, events chan<- agent.AgentEvent, submit <-chan string) {
	for {
		a := agent.New(llmClient, exec)
		go func() {
			for ev := range a.Events() {
				events <- ev
			}
		}()
		a.Run(*messages)

		prompt, ok := <-submit
		if !ok {
			close(events)
			return
		}
		if prompt == "" {
			close(events)
			return
		}
		*messages = append(*messages, core.Message{Role: "user", Content: prompt})
	}
}

func init() {
	chatCmd.Flags().StringVarP(&modelFlag, "model", "m", "deepseek-v4-flash", "Model to use (deepseek-v4-flash or deepseek-v4-pro)")
	rootCmd.AddCommand(chatCmd)
}
