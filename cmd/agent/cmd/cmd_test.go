package cmd

import (
	"testing"
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
	if cmd.Use != "run" {
		t.Fatalf("expected Use='run', got %q", cmd.Use)
	}
}
