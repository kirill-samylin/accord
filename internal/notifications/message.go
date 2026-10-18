package notifications

import (
	"agent-board/internal/interchange"
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"time"
)

const MaxBody = 32 << 10

type Task struct {
	Key        string   `json:"key"`
	Title      string   `json:"title"`
	Role       string   `json:"role"`
	Category   string   `json:"category,omitempty"`
	Complexity string   `json:"complexity,omitempty"`
	Eligible   []string `json:"eligible,omitempty"`
	URL        string   `json:"url,omitempty"`
}
type Epic struct {
	Key   string `json:"key"`
	Title string `json:"title"`
}
type Payload struct {
	DeliveryID  string    `json:"delivery_id"`
	EventCursor int64     `json:"event_cursor"`
	Event       string    `json:"event"`
	TS          time.Time `json:"ts"`
	Task        *Task     `json:"task,omitempty"`
	Epic        *Epic     `json:"epic,omitempty"`
}
type Message struct {
	URL    string
	Body   []byte
	Secret []byte
}

func Encode(p Payload) ([]byte, error) {
	switch p.Event {
	case "task_created_for_human", "escalated", "lease_expired", "epic_done":
	default:
		return nil, errors.New("unsupported notification event")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`).MatchString(p.DeliveryID) || p.EventCursor <= 0 || p.TS.IsZero() {
		return nil, errors.New("invalid notification identity")
	}
	// A task URL requires an independently trusted browser origin. V1 has none;
	// callers cannot accidentally leak a URL inferred from a request Host header.
	if p.Task != nil && p.Task.URL != "" {
		return nil, errors.New("browser origin not configured")
	}
	raw, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	if len(raw) > MaxBody {
		return nil, errors.New("notification body limit")
	}
	return raw, nil
}

// NewRequest signs the exact stored bytes on each attempt with the current
// secret. The caller keeps delivery_id/ts/body stable across retries.
func NewRequest(ctx context.Context, m Message) (*http.Request, error) {
	u, err := ValidateURL(m.URL)
	if err != nil {
		return nil, err
	}
	if len(m.Secret) < 32 || len(m.Body) > MaxBody || interchange.CheckJSON(m.Body, 16) != nil {
		return nil, errors.New("invalid notification")
	}
	var p Payload
	if interchange.DecodeStrict(m.Body, &p) != nil {
		return nil, errors.New("invalid notification")
	}
	if _, err := Encode(p); err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(m.Body))
	if err != nil {
		return nil, errors.New("invalid notification")
	}
	mac := hmac.New(sha256.New, m.Secret)
	mac.Write(m.Body)
	req.Header.Set("X-Agent-Board-Signature", "sha256="+hex.EncodeToString(mac.Sum(nil)))
	req.Header.Set("X-Agent-Board-Delivery", p.DeliveryID)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept-Encoding", "identity")
	return req, nil
}
