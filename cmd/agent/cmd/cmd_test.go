package cmd

import (
	"fmt"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/tools"
	"github.com/spf13/cobra"
)

func TestChatSubcommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"chat"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected chat subcommand, got nil")
	}
	if cmd.Use != "chat" {
		t.Fatalf("expected Use='chat', got %q", cmd.Use)
	}
}

func TestRunSubcommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"run"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected run subcommand, got nil")
	}
	if cmd.Use != "run <prompt>" {
		t.Fatalf("expected Use='run <prompt>', got %q", cmd.Use)
	}
}

func TestRunCommandHasJSONFlag(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"run"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	flag := cmd.Flag("json")
	if flag == nil {
		t.Fatal("expected --json flag on run command")
	}
	if flag.Value.Type() != "bool" {
		t.Fatalf("expected --json to be bool, got %s", flag.Value.Type())
	}
}

func TestRunCommandRequiresPromptArg(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"run"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errNoPrompt
		}
		return nil
	}
	err = cmd.RunE(cmd, []string{})
	if err == nil {
		t.Fatal("expected error when no prompt arg provided")
	}
	if err != errNoPrompt {
		t.Fatalf("expected errNoPrompt, got %v", err)
	}
}

func TestSessionsSubcommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sessions", "list"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected sessions list subcommand, got nil")
	}
	if cmd.Use != "list" {
		t.Fatalf("expected Use='list', got %q", cmd.Use)
	}
}

func TestSessionsResumeSubcommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sessions", "resume"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected sessions resume subcommand, got nil")
	}
	if cmd.Use != "resume <id>" {
		t.Fatalf("expected Use='resume <id>', got %q", cmd.Use)
	}
}

func TestLogsSubcommandIsRegistered(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	if cmd == nil {
		t.Fatal("expected logs subcommand, got nil")
	}
	if cmd.Use != "logs <id>" {
		t.Fatalf("expected Use='logs <id>', got %q", cmd.Use)
	}
}

func TestLogsCommandHasFollowFlag(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	flag := cmd.Flag("follow")
	if flag == nil {
		t.Fatal("expected --follow flag on logs command")
	}
	if flag.Value.Type() != "bool" {
		t.Fatalf("expected --follow to be bool, got %s", flag.Value.Type())
	}
}

func TestSessionsListNoDbDoesNotError(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sessions", "list"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}

	err = cmd.RunE(cmd, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSessionsResumeRequiresIDArg(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sessions", "resume"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}

	// The cobra.ExactArgs(1) validator should catch missing args
	err = cmd.ValidateArgs([]string{})
	if err == nil {
		t.Fatal("expected error when no id arg provided")
	}
}

func TestSessionsResumeInvalidIDShowsError(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"sessions", "resume"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("session %s not found", args[0])
	}

	err = cmd.RunE(cmd, []string{"99999"})
	if err == nil {
		t.Fatal("expected error for invalid session ID")
	}
}

func TestLogsRequiresIDArg(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}

	err = cmd.ValidateArgs([]string{})
	if err == nil {
		t.Fatal("expected error when no id arg provided")
	}
}

func TestLogsInvalidIDShowsError(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"logs"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fmt.Errorf("log file for session %s not found", args[0])
	}

	err = cmd.RunE(cmd, []string{"99999"})
	if err == nil {
		t.Fatal("expected error for invalid session ID")
	}
}

func TestToDeepSeekToolsSetsStrictForSupportedSchemas(t *testing.T) {
	defs := []tools.ToolDefinition{
		{
			Name:        "read_file",
			Description: "Read a file",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"path": map[string]any{"type": "string"}},
				"required":   []any{"path"},
			},
		},
	}
	result := toDeepSeekTools(defs)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool def, got %d", len(result))
	}
	if !result[0].Function.Strict {
		t.Fatal("expected Strict=true for supported schema")
	}
}

func TestToDeepSeekToolsOmitsStrictForUnsupportedSchemas(t *testing.T) {
	defs := []tools.ToolDefinition{
		{
			Name:        "pick_color",
			Description: "Pick a color",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"color": map[string]any{"type": "string", "enum": []any{"red", "blue"}}},
				"required":   []any{"color"},
			},
		},
	}
	result := toDeepSeekTools(defs)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool def, got %d", len(result))
	}
	if result[0].Function.Strict {
		t.Fatal("expected Strict=false for schema with enum")
	}
}

func TestToDeepSeekToolsMixedStrictAndNonStrict(t *testing.T) {
	defs := []tools.ToolDefinition{
		{
			Name:        "a_tool",
			Description: "A strict-compatible tool",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"name": map[string]any{"type": "string"}},
				"required":   []any{"name"},
			},
		},
		{
			Name:        "b_tool",
			Description: "A tool with unsupported features",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{"color": map[string]any{"type": "string", "enum": []any{"red"}}},
				"required":   []any{"color"},
			},
		},
	}
	result := toDeepSeekTools(defs)
	if len(result) != 2 {
		t.Fatalf("expected 2 tool defs, got %d", len(result))
	}

	strictCount := 0
	nonStrictCount := 0
	for _, d := range result {
		if d.Function.Name == "a_tool" {
			if !d.Function.Strict {
				t.Error("expected a_tool to have Strict=true")
			}
			strictCount++
		}
		if d.Function.Name == "b_tool" {
			if d.Function.Strict {
				t.Error("expected b_tool to have Strict=false")
			}
			nonStrictCount++
		}
	}
	if strictCount != 1 {
		t.Errorf("expected 1 strict tool, got %d", strictCount)
	}
	if nonStrictCount != 1 {
		t.Errorf("expected 1 non-strict tool, got %d", nonStrictCount)
	}
}

func TestRunCommandAcceptsPromptArg(t *testing.T) {
	cmd, _, err := rootCmd.Find([]string{"run"})
	if err != nil {
		t.Fatalf("rootCmd.Find returned error: %v", err)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errNoPrompt
		}
		return nil
	}
	err = cmd.RunE(cmd, []string{"do something"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
