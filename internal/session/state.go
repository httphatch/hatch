package session

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// persistedSession is the JSON-serializable form of a session.
type persistedSession struct {
	Project    string         `json:"project"`
	Name       string         `json:"name"`
	Ports      map[string]int `json:"ports"`
	CreatedAt  time.Time      `json:"created_at"`
	LastActive time.Time      `json:"last_active"`
	TTLSeconds int            `json:"ttl_seconds"`
}

func saveState(path string, sessions map[SessionID]*Session) error {
	persisted := make([]persistedSession, 0, len(sessions))
	for _, s := range sessions {
		if s.State != StateRunning {
			continue
		}
		persisted = append(persisted, persistedSession{
			Project:    s.ID.Project,
			Name:       s.ID.Name,
			Ports:      s.Ports,
			CreatedAt:  s.CreatedAt,
			LastActive: s.LastActive,
			TTLSeconds: int(s.TTL.Seconds()),
		})
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}
	return os.WriteFile(path, data, 0o600)
}

func loadState(path string) ([]persistedSession, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read session state: %w", err)
	}

	var sessions []persistedSession
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("unmarshal session state: %w", err)
	}
	return sessions, nil
}
