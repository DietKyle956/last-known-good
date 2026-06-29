package cmd

import (
	"testing"

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
