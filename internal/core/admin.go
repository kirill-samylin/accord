package core

import "time"

// TokenInfo is safe for local administration: it never exposes credential material.
type TokenInfo struct {
	ID        string    `json:"id"`
	Owner     string    `json:"owner"`
	Machine   string    `json:"machine"`
	ExpiresAt time.Time `json:"expires_at"`
	Revoked   bool      `json:"revoked"`
}

type SessionInfo struct {
	SessionID string    `json:"session_id"`
	Owner     string    `json:"owner"`
	Machine   string    `json:"machine"`
	Role      string    `json:"role"`
	RunID     string    `json:"run_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

type RecoveryResult struct {
	RevokedTokens int `json:"revoked_tokens"`
	Sessions      int `json:"sessions"`
	Leases        int `json:"leases"`
}

type BackupResult struct {
	SHA256        string    `json:"sha256"`
	CreatedAt     time.Time `json:"created_at"`
	SchemaVersion int       `json:"schema_version"`
}
