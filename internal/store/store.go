package store

import (
	"database/sql"
	"fmt"

	"github.com/DietKyle956/last-known-good/internal/core"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func New(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

type SessionRecord struct {
	ID        int64
	CreatedAt string
}

type MessageRecord struct {
	ID        int64
	SessionID int64
	Role      string
	Content   string
	Model     string
	Ordinal   int
}

func (s *Store) SaveMessage(sessionID int64, role, content, model string) error {
	var maxOrd sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(ordinal) FROM messages WHERE session_id = ?", sessionID).Scan(&maxOrd)
	if err != nil {
		return fmt.Errorf("query max ordinal: %w", err)
	}
	ord := int64(0)
	if maxOrd.Valid {
		ord = maxOrd.Int64 + 1
	}
	_, err = s.db.Exec(
		"INSERT INTO messages (session_id, role, content, model, ordinal) VALUES (?, ?, ?, ?, ?)",
		sessionID, role, content, model, ord,
	)
	if err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

func (s *Store) GetMessages(sessionID int64) ([]MessageRecord, error) {
	rows, err := s.db.Query(
		"SELECT id, session_id, role, content, model, ordinal FROM messages WHERE session_id = ? ORDER BY ordinal",
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	var msgs []MessageRecord
	for rows.Next() {
		var m MessageRecord
		if err := rows.Scan(&m.ID, &m.SessionID, &m.Role, &m.Content, &m.Model, &m.Ordinal); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}

func (s *Store) SaveMessages(sessionID int64, messages []core.Message) error {
	for i, m := range messages {
		if err := s.SaveMessage(sessionID, m.Role, m.Content, ""); err != nil {
			return fmt.Errorf("save message %d: %w", i, err)
		}
	}
	return nil
}

func (s *Store) Resume(sessionID int64) ([]core.Message, error) {
	exists, err := s.SessionExists(sessionID)
	if err != nil {
		return nil, fmt.Errorf("check session: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("session %d not found", sessionID)
	}

	records, err := s.GetMessages(sessionID)
	if err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}

	messages := make([]core.Message, len(records))
	for i, r := range records {
		messages[i] = core.Message{
			Role:    r.Role,
			Content: r.Content,
		}
	}

	return messages, nil
}

func (s *Store) SaveHookEvent(sessionID int64, eventType, payload string) error {
	var maxOrd sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(ordinal) FROM hook_events WHERE session_id = ?", sessionID).Scan(&maxOrd)
	if err != nil {
		return fmt.Errorf("query max ordinal: %w", err)
	}
	ord := int64(0)
	if maxOrd.Valid {
		ord = maxOrd.Int64 + 1
	}
	_, err = s.db.Exec(
		"INSERT INTO hook_events (session_id, event_type, payload, ordinal) VALUES (?, ?, ?, ?)",
		sessionID, eventType, payload, ord,
	)
	if err != nil {
		return fmt.Errorf("save hook event: %w", err)
	}
	return nil
}

func (s *Store) SaveToolCall(sessionID int64, name, arguments, result string, isError bool, durationMs int64) error {
	var maxOrd sql.NullInt64
	err := s.db.QueryRow("SELECT MAX(ordinal) FROM tool_calls WHERE session_id = ?", sessionID).Scan(&maxOrd)
	if err != nil {
		return fmt.Errorf("query max ordinal: %w", err)
	}
	ord := int64(0)
	if maxOrd.Valid {
		ord = maxOrd.Int64 + 1
	}
	isErr := 0
	if isError {
		isErr = 1
	}
	_, err = s.db.Exec(
		"INSERT INTO tool_calls (session_id, name, arguments, result, is_error, duration_ms, ordinal) VALUES (?, ?, ?, ?, ?, ?, ?)",
		sessionID, name, arguments, result, isErr, durationMs, ord,
	)
	if err != nil {
		return fmt.Errorf("save tool call: %w", err)
	}
	return nil
}

func (s *Store) SessionExists(id int64) (bool, error) {
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM sessions WHERE id = ?", id).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("query session: %w", err)
	}
	return count > 0, nil
}

func (s *Store) ListSessions() ([]SessionRecord, error) {
	rows, err := s.db.Query("SELECT id, created_at FROM sessions ORDER BY id DESC")
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []SessionRecord
	for rows.Next() {
		var sess SessionRecord
		if err := rows.Scan(&sess.ID, &sess.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, sess)
	}
	if sessions == nil {
		sessions = []SessionRecord{}
	}
	return sessions, rows.Err()
}

func (s *Store) CreateSession() (int64, error) {
	res, err := s.db.Exec("INSERT INTO sessions DEFAULT VALUES")
	if err != nil {
		return 0, fmt.Errorf("create session: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("last insert id: %w", err)
	}
	return id, nil
}

func migrate(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS sessions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at TEXT NOT NULL DEFAULT (datetime('now'))
	);

	CREATE TABLE IF NOT EXISTS messages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL REFERENCES sessions(id),
		role TEXT NOT NULL,
		content TEXT NOT NULL DEFAULT '',
		model TEXT NOT NULL DEFAULT '',
		ordinal INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tool_calls (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL REFERENCES sessions(id),
		name TEXT NOT NULL,
		arguments TEXT NOT NULL DEFAULT '',
		result TEXT NOT NULL DEFAULT '',
		is_error INTEGER NOT NULL DEFAULT 0,
		duration_ms INTEGER NOT NULL DEFAULT 0,
		ordinal INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS hook_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id INTEGER NOT NULL REFERENCES sessions(id),
		event_type TEXT NOT NULL,
		payload TEXT NOT NULL DEFAULT '',
		ordinal INTEGER NOT NULL
	);
	`
	_, err := db.Exec(schema)
	return err
}
