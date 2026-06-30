package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/DietKyle956/last-known-good/internal/agent"
	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/DietKyle956/last-known-good/internal/skills"
	"github.com/DietKyle956/last-known-good/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var modelFlag string

type stubExecutor struct{}

func (s *stubExecutor) Execute(_ context.Context, call core.ToolCall) core.ToolResult {
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

		loader := skills.NewLoader("skills")
		if err := loader.Load(); err != nil {
			return fmt.Errorf("load skills: %w", err)
		}
		var skillSummaries string
		if ss := loader.Summaries(); len(ss) > 0 {
			var b strings.Builder
			for _, s := range ss {
				b.WriteString("- **" + s.Name + "**: " + s.Description + "\n")
			}
			skillSummaries = b.String()
		}

		messages := []core.Message{
			{Role: "system", Content: core.BuildSystemPrompt(skillSummaries, "")},
		}

		return runTUI(client, messages)
	},
}

func runTUI(llmClient agent.LLM, messages []core.Message) error {
	submit := make(chan string, 64)
	events := make(chan agent.AgentEvent, 128)
	exec := &stubExecutor{}

	sess := agent.NewSession(llmClient, exec)
	ctx := context.Background()
	go sess.Run(ctx, messages, submit, events)

	m := tui.New(events, submit)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func init() {
	chatCmd.Flags().StringVarP(&modelFlag, "model", "m", "deepseek-v4-flash", "Model to use (deepseek-v4-flash or deepseek-v4-pro)")
	rootCmd.AddCommand(chatCmd)
}
