package session

import (
	"path/filepath"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/store"
)

func TestResumeReturnsMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	messages := []struct {
		role    string
		content string
		model   string
	}{
		{"system", "You are a helpful assistant.", "deepseek-v4-flash"},
		{"user", "Hello!", ""},
		{"assistant", "Hi there!", "deepseek-v4-flash"},
	}

	for _, m := range messages {
		if err := s.SaveMessage(sessionID, m.role, m.content, m.model); err != nil {
			t.Fatalf("SaveMessage() returned error: %v", err)
		}
	}

	got, err := Resume(s, sessionID)
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if len(got) != len(messages) {
		t.Fatalf("expected %d messages, got %d", len(messages), len(got))
	}

	for i, m := range messages {
		if got[i].Role != m.role {
			t.Errorf("message %d: expected role %q, got %q", i, m.role, got[i].Role)
		}
		if got[i].Content != m.content {
			t.Errorf("message %d: expected content %q, got %q", i, m.content, got[i].Content)
		}
	}
}

func TestResumeNonExistentSessionReturnsError(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	_, err = Resume(s, 99999)
	if err == nil {
		t.Fatal("Resume(99999) expected error, got nil")
	}
}

func TestSaveMessagesToSession(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	original := []core.Message{
		{Role: "user", Content: "Hello"},
		{Role: "assistant", Content: "Hi!"},
	}

	if err := SaveMessages(s, sessionID, original); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	more := []core.Message{
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm good!"},
	}

	if err := SaveMessages(s, sessionID, more); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	got, err := Resume(s, sessionID)
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	expected := append(original, more...)
	if len(got) != len(expected) {
		t.Fatalf("expected %d messages, got %d", len(expected), len(got))
	}

	for i, m := range expected {
		if got[i].Role != m.Role {
			t.Errorf("message %d: expected role %q, got %q", i, m.Role, got[i].Role)
		}
		if got[i].Content != m.Content {
			t.Errorf("message %d: expected content %q, got %q", i, m.Content, got[i].Content)
		}
	}
}

func TestRoundTripStateEquivalence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	original := []core.Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "What is Go?"},
		{Role: "assistant", Content: "Go is a programming language."},
		{Role: "user", Content: "Thanks!"},
	}

	if err := SaveMessages(s, sessionID, original); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	got, err := Resume(s, sessionID)
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if len(got) != len(original) {
		t.Fatalf("expected %d messages, got %d", len(original), len(got))
	}

	for i := range original {
		if got[i].Role != original[i].Role {
			t.Errorf("message %d: role mismatch: %q vs %q", i, got[i].Role, original[i].Role)
		}
		if got[i].Content != original[i].Content {
			t.Errorf("message %d: content mismatch: %q vs %q", i, got[i].Content, original[i].Content)
		}
	}
}

func TestResumeMessagesAreCoreMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	if err := s.SaveMessage(sessionID, "user", "Hello", ""); err != nil {
		t.Fatalf("SaveMessage() returned error: %v", err)
	}

	got, err := Resume(s, sessionID)
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got))
	}

	var _ core.Message = got[0]
}
