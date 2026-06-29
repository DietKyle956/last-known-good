package session

import (
	"fmt"

	"github.com/DietKyle956/last-known-good/internal/core"
	"github.com/DietKyle956/last-known-good/internal/store"
)

func SaveMessages(s *store.Store, sessionID int64, messages []core.Message) error {
	for i, m := range messages {
		if err := s.SaveMessage(sessionID, m.Role, m.Content, ""); err != nil {
			return fmt.Errorf("save message %d: %w", i, err)
		}
	}
	return nil
}

func Resume(s *store.Store, sessionID int64) ([]core.Message, error) {
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
