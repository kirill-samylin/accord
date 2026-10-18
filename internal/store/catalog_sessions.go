package store

import (
	"context"
	"encoding/json"
)

func sessionOrders(ctx context.Context, b *board) (map[string]int64, error) {
	result := map[string]int64{}
	rows, err := b.tx.QueryContext(ctx, `SELECT cursor,data FROM events WHERE json_extract(data,'$.type')='session_opened' ORDER BY cursor`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var cursor int64
		var raw []byte
		if err = rows.Scan(&cursor, &raw); err != nil {
			return nil, err
		}
		var e struct {
			SessionID string `json:"session_id"`
		}
		if err = json.Unmarshal(raw, &e); err != nil {
			return nil, err
		}
		result[e.SessionID] = cursor
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	for _, e := range b.Events {
		if e.Type == "session_opened" {
			result[e.SessionID] = e.Cursor
		}
	}
	return result, nil
}
