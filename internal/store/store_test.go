package store

import (
	"path/filepath"
	"testing"

	"github.com/DietKyle956/last-known-good/internal/core"
)

func TestSchemaAppliedToNewDB(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	rows, err := s.db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("query sqlite_master failed: %v", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		tables = append(tables, name)
	}

	expected := []string{"hook_events", "messages", "sessions", "tool_calls"}
	for _, e := range expected {
		found := false
		for _, got := range tables {
			if got == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected table %q not found in %v", e, tables)
		}
	}
}

func TestSaveAndGetMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	msgs := []struct {
		role    string
		content string
		model   string
	}{
		{"user", "Hello", ""},
		{"assistant", "Hi there!", "deepseek-v4-flash"},
		{"user", "What's the weather?", ""},
	}

	for i, m := range msgs {
		err := s.SaveMessage(sessionID, m.role, m.content, m.model)
		if err != nil {
			t.Fatalf("SaveMessage(%d) returned error: %v", i, err)
		}
	}

	got, err := s.GetMessages(sessionID)
	if err != nil {
		t.Fatalf("GetMessages() returned error: %v", err)
	}

	if len(got) != len(msgs) {
		t.Fatalf("expected %d messages, got %d", len(msgs), len(got))
	}

	for i, m := range msgs {
		if got[i].Role != m.role {
			t.Errorf("message %d: expected role %q, got %q", i, m.role, got[i].Role)
		}
		if got[i].Content != m.content {
			t.Errorf("message %d: expected content %q, got %q", i, m.content, got[i].Content)
		}
		if got[i].Model != m.model {
			t.Errorf("message %d: expected model %q, got %q", i, m.model, got[i].Model)
		}
	}
}

func TestSaveToolCall(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	tcs := []struct {
		name       string
		args       string
		result     string
		isError    bool
		durationMs int64
	}{
		{"read_file", `{"path":"test.txt"}`, "file contents", false, 5},
		{"bash", `{"cmd":"ls"}`, "file1.txt\nfile2.txt", false, 42},
		{"write_file", `{"path":"x.txt","content":"data"}`, "", true, 10},
	}

	for i, tc := range tcs {
		err := s.SaveToolCall(sessionID, tc.name, tc.args, tc.result, tc.isError, tc.durationMs)
		if err != nil {
			t.Fatalf("SaveToolCall(%d) returned error: %v", i, err)
		}
	}

	rows, err := s.db.Query(
		"SELECT name, arguments, result, is_error, duration_ms FROM tool_calls WHERE session_id = ? ORDER BY ordinal",
		sessionID,
	)
	if err != nil {
		t.Fatalf("query tool calls failed: %v", err)
	}
	defer rows.Close()

	var idx int
	for rows.Next() {
		if idx >= len(tcs) {
			t.Fatal("more tool calls in DB than expected")
		}
		var name, args, result string
		var isError int
		var durationMs int64
		if err := rows.Scan(&name, &args, &result, &isError, &durationMs); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		expected := tcs[idx]
		if name != expected.name {
			t.Errorf("tool call %d: expected name %q, got %q", idx, expected.name, name)
		}
		if args != expected.args {
			t.Errorf("tool call %d: expected args %q, got %q", idx, expected.args, args)
		}
		if result != expected.result {
			t.Errorf("tool call %d: expected result %q, got %q", idx, expected.result, result)
		}
		if (isError != 0) != expected.isError {
			t.Errorf("tool call %d: expected isError=%v, got isError=%d", idx, expected.isError, isError)
		}
		if durationMs != expected.durationMs {
			t.Errorf("tool call %d: expected durationMs=%d, got %d", idx, expected.durationMs, durationMs)
		}
		idx++
	}
	if idx != len(tcs) {
		t.Errorf("expected %d tool calls, got %d", len(tcs), idx)
	}
}

func TestSaveHookEvent(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	events := []struct {
		eventType string
		payload   string
	}{
		{"tool_call_started", `{"tool":"read_file"}`},
		{"tool_call_finished", `{"tool":"read_file","duration_ms":5}`},
		{"turn_complete", `{}`},
	}

	for i, e := range events {
		err := s.SaveHookEvent(sessionID, e.eventType, e.payload)
		if err != nil {
			t.Fatalf("SaveHookEvent(%d) returned error: %v", i, err)
		}
	}

	rows, err := s.db.Query(
		"SELECT event_type, payload FROM hook_events WHERE session_id = ? ORDER BY ordinal",
		sessionID,
	)
	if err != nil {
		t.Fatalf("query hook events failed: %v", err)
	}
	defer rows.Close()

	var idx int
	for rows.Next() {
		if idx >= len(events) {
			t.Fatal("more hook events in DB than expected")
		}
		var eventType, payload string
		if err := rows.Scan(&eventType, &payload); err != nil {
			t.Fatalf("scan failed: %v", err)
		}
		expected := events[idx]
		if eventType != expected.eventType {
			t.Errorf("event %d: expected type %q, got %q", idx, expected.eventType, eventType)
		}
		if payload != expected.payload {
			t.Errorf("event %d: expected payload %q, got %q", idx, expected.payload, payload)
		}
		idx++
	}
	if idx != len(events) {
		t.Errorf("expected %d hook events, got %d", len(events), idx)
	}
}

func TestDurabilityCloseAndReopen(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}

	sessionID, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	err = s.SaveMessage(sessionID, "user", "Hello", "")
	if err != nil {
		t.Fatalf("SaveMessage() returned error: %v", err)
	}

	err = s.SaveToolCall(sessionID, "read_file", `{"path":"x"}`, "content", false, 5)
	if err != nil {
		t.Fatalf("SaveToolCall() returned error: %v", err)
	}

	err = s.SaveHookEvent(sessionID, "tool_call_started", `{"tool":"read_file"}`)
	if err != nil {
		t.Fatalf("SaveHookEvent() returned error: %v", err)
	}

	s.Close()

	s2, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) after close returned error: %v", dbPath, err)
	}
	defer s2.Close()

	msgs, err := s2.GetMessages(sessionID)
	if err != nil {
		t.Fatalf("GetMessages() returned error: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "Hello" {
		t.Errorf("unexpected message: %+v", msgs[0])
	}

	var toolCount int
	err = s2.db.QueryRow("SELECT COUNT(*) FROM tool_calls WHERE session_id = ?", sessionID).Scan(&toolCount)
	if err != nil {
		t.Fatalf("query tool calls: %v", err)
	}
	if toolCount != 1 {
		t.Errorf("expected 1 tool call, got %d", toolCount)
	}

	var hookCount int
	err = s2.db.QueryRow("SELECT COUNT(*) FROM hook_events WHERE session_id = ?", sessionID).Scan(&hookCount)
	if err != nil {
		t.Fatalf("query hook events: %v", err)
	}
	if hookCount != 1 {
		t.Errorf("expected 1 hook event, got %d", hookCount)
	}
}

func TestMultipleSessionsIsolated(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	s1, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}
	s2, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	err = s.SaveMessage(s1, "user", "Session 1 message", "")
	if err != nil {
		t.Fatalf("SaveMessage s1: %v", err)
	}
	err = s.SaveMessage(s2, "user", "Session 2 message", "")
	if err != nil {
		t.Fatalf("SaveMessage s2: %v", err)
	}

	msgs1, err := s.GetMessages(s1)
	if err != nil {
		t.Fatalf("GetMessages s1: %v", err)
	}
	if len(msgs1) != 1 || msgs1[0].Content != "Session 1 message" {
		t.Errorf("unexpected messages for session 1: %+v", msgs1)
	}

	msgs2, err := s.GetMessages(s2)
	if err != nil {
		t.Fatalf("GetMessages s2: %v", err)
	}
	if len(msgs2) != 1 || msgs2[0].Content != "Session 2 message" {
		t.Errorf("unexpected messages for session 2: %+v", msgs2)
	}
}

func TestSessionExists(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	id, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	exists, err := s.SessionExists(id)
	if err != nil {
		t.Fatalf("SessionExists(%d) returned error: %v", id, err)
	}
	if !exists {
		t.Errorf("SessionExists(%d) = false, want true", id)
	}

	exists, err = s.SessionExists(99999)
	if err != nil {
		t.Fatalf("SessionExists(99999) returned error: %v", err)
	}
	if exists {
		t.Errorf("SessionExists(99999) = true, want false")
	}
}

func TestCreateSession(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	id, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}
	if id <= 0 {
		t.Errorf("expected positive session ID, got %d", id)
	}

	var count int
	err = s.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", id).Scan(&count)
	if err != nil {
		t.Fatalf("query session count failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 session, got %d", count)
	}
}

// --- Session package tests moved here ---

func TestResumeReturnsMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
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

	got, err := s.Resume(sessionID)
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
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	_, err = s.Resume(99999)
	if err == nil {
		t.Fatal("Resume(99999) expected error, got nil")
	}
}

func TestSaveMessagesToSession(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
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

	if err := s.SaveMessages(sessionID, original); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	more := []core.Message{
		{Role: "user", Content: "How are you?"},
		{Role: "assistant", Content: "I'm good!"},
	}

	if err := s.SaveMessages(sessionID, more); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	got, err := s.Resume(sessionID)
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
	s, err := New(dbPath)
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

	if err := s.SaveMessages(sessionID, original); err != nil {
		t.Fatalf("SaveMessages() returned error: %v", err)
	}

	got, err := s.Resume(sessionID)
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

func TestListSessionsReturnsAllSessions(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	s1, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}
	s2, err := s.CreateSession()
	if err != nil {
		t.Fatalf("CreateSession() returned error: %v", err)
	}

	sessions, err := s.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}

	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}

	if sessions[0].ID != s2 || sessions[0].CreatedAt == "" {
		t.Errorf("expected first session to be %d with non-empty created_at, got %+v", s2, sessions[0])
	}
	if sessions[1].ID != s1 || sessions[1].CreatedAt == "" {
		t.Errorf("expected second session to be %d with non-empty created_at, got %+v", s1, sessions[1])
	}
}

func TestListSessionsEmptyReturnsEmptySlice(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New(%q) returned error: %v", dbPath, err)
	}
	defer s.Close()

	sessions, err := s.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() returned error: %v", err)
	}

	if sessions == nil {
		t.Fatal("ListSessions() returned nil, expected empty slice")
	}
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestResumeMessagesAreCoreMessages(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	s, err := New(dbPath)
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

	got, err := s.Resume(sessionID)
	if err != nil {
		t.Fatalf("Resume() returned error: %v", err)
	}

	if len(got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got))
	}

	var _ core.Message = got[0]
}
