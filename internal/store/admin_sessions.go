package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
)

// ListSessions is a local-only view. Reading it never renews a session.
func (s *Store) ListSessions(ctx context.Context) ([]core.SessionInfo, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT sessions.data, tokens.owner FROM sessions JOIN tokens ON tokens.id=json_extract(sessions.data,'$.principal_id') ORDER BY sessions.key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []core.SessionInfo{}
	for rows.Next() {
		var raw []byte
		var owner string
		if err = rows.Scan(&raw, &owner); err != nil {
			return nil, err
		}
		var session core.Session
		if err = json.Unmarshal(raw, &session); err != nil {
			return nil, err
		}
		result = append(result, core.SessionInfo{SessionID: session.SessionID, Owner: owner, Machine: session.Machine, Role: session.Role, RunID: session.RunID, ExpiresAt: session.ExpiresAt})
	}
	return result, rows.Err()
}
