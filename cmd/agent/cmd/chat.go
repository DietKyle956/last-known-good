package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/spf13/cobra"
)

var modelFlag string

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

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("> ")
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				fmt.Print("> ")
				continue
			}
			if line == "/quit" || line == "/exit" {
				break
			}

			messages = append(messages, core.Message{Role: "user", Content: line})

			results, err := client.Chat(messages)
			if err != nil {
				return fmt.Errorf("chat error: %w", err)
			}

			var fullResponse strings.Builder
			for r := range results {
				if r.Err != nil {
					return r.Err
				}
				if r.IsChunk {
					fullResponse.WriteString(r.Content)
					fmt.Print(r.Content)
				}
			}

			fmt.Println()
			messages = append(messages, core.Message{Role: "assistant", Content: fullResponse.String()})
			fmt.Print("> ")
		}

		return nil
	},
}

func init() {
	chatCmd.Flags().StringVarP(&modelFlag, "model", "m", "deepseek-v4-flash", "Model to use (deepseek-v4-flash or deepseek-v4-pro)")
	rootCmd.AddCommand(chatCmd)
}
