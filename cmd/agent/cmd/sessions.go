package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/DietKyle956/last-known-good/internal/llm"
	"github.com/DietKyle956/last-known-good/internal/store"
	"github.com/spf13/cobra"
)

var sessionsCmd = &cobra.Command{
	Use:   "sessions",
	Short: "Manage past sessions",
}

var sessionsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List past sessions",
	RunE: func(cmd *cobra.Command, args []string) error {
		dbPath := cmd.Flag("db").Value.String()
		s, err := store.New(dbPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, "no sessions found")
			return nil
		}
		defer s.Close()

		sessions, err := s.ListSessions()
		if err != nil {
			return fmt.Errorf("list sessions: %w", err)
		}

		if len(sessions) == 0 {
			fmt.Println("no sessions found")
			return nil
		}

		for _, sess := range sessions {
			fmt.Printf("%d\t%s\n", sess.ID, sess.CreatedAt)
		}
		return nil
	},
}

var sessionsResumeCmd = &cobra.Command{
	Use:   "resume <id>",
	Short: "Resume a past session",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid session ID: %s", args[0])
		}

		dbPath := cmd.Flag("db").Value.String()
		s, err := store.New(dbPath)
		if err != nil {
			return fmt.Errorf("open store: %w", err)
		}
		defer s.Close()

		messages, err := s.Resume(id)
		if err != nil {
			return fmt.Errorf("resume session %d: %w", id, err)
		}

		apiKey := llm.APIKeyForModel(modelFlag)
		if apiKey == "" {
			return fmt.Errorf("no API key found for model %q — set DEEPSEEK_FLASH_KEY, DEEPSEEK_PRO_KEY, or DEEPSEEK_API_KEY", modelFlag)
		}

		client := llm.NewDeepSeekClient(llm.DeepSeekConfig{
			APIKey: apiKey,
			Model:  modelFlag,
			Stream: true,
		})

		return runTUI(client, messages)
	},
}

func init() {
	cwd, _ := os.Getwd()
	defaultDB := filepath.Join(cwd, "lkg.db")

	sessionsCmd.PersistentFlags().String("db", defaultDB, "Path to the session store database")
	sessionsCmd.AddCommand(sessionsListCmd)
	sessionsCmd.AddCommand(sessionsResumeCmd)
	rootCmd.AddCommand(sessionsCmd)
}
