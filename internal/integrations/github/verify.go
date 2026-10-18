// Package github verifies and normalizes webhook facts without mutating a board.
package github

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"mime"
	"net/http"
	"regexp"
	"strings"
)

const MaxBody = 1 << 20

type Error struct {
	Status int
	Code   string
}

func (e *Error) Error() string { return e.Code }

type Envelope struct {
	DeliveryID string
	Event      string
	BodySHA256 string
	Raw        []byte
}

var guid = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// SingleHeader also rejects manually constructed maps with mixed-case duplicate
// names, rather than relying on callers to have canonicalized their headers.
func SingleHeader(h http.Header, name string) (string, bool) {
	var values []string
	for k, v := range h {
		if strings.EqualFold(k, name) {
			values = append(values, v...)
		}
	}
	if len(values) != 1 || values[0] == "" {
		return "", false
	}
	return values[0], true
}
func Verify(h http.Header, raw, secret []byte) (Envelope, error) {
	var empty Envelope
	if len(raw) > MaxBody {
		return empty, &Error{413, "BODY_LIMIT"}
	}
	encoding, has := SingleHeader(h, "Content-Encoding")
	for k := range h {
		if strings.EqualFold(k, "Content-Encoding") && (!has || encoding != "identity") {
			return empty, &Error{415, "ENCODING"}
		}
	}
	contentType, ok := SingleHeader(h, "Content-Type")
	media, _, parseErr := mime.ParseMediaType(contentType)
	if !ok || parseErr != nil || media != "application/json" {
		return empty, &Error{415, "CONTENT_TYPE"}
	}
	signature, ok := SingleHeader(h, "X-Hub-Signature-256")
	if !ok || len(secret) < 32 || len(signature) != 71 || !strings.HasPrefix(signature, "sha256=") {
		return empty, &Error{401, "SIGNATURE"}
	}
	supplied, err := hex.DecodeString(signature[7:])
	if err != nil {
		return empty, &Error{401, "SIGNATURE"}
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write(raw)
	if !hmac.Equal(supplied, mac.Sum(nil)) {
		return empty, &Error{401, "SIGNATURE"}
	}
	id, ok := SingleHeader(h, "X-GitHub-Delivery")
	if !ok || !guid.MatchString(id) {
		return empty, &Error{422, "DELIVERY_ID"}
	}
	event, ok := SingleHeader(h, "X-GitHub-Event")
	if !ok || len(event) > 64 || !regexp.MustCompile(`^[a-z_]+$`).MatchString(event) {
		return empty, &Error{422, "EVENT"}
	}
	sum := sha256.Sum256(raw)
	return Envelope{strings.ToLower(id), event, hex.EncodeToString(sum[:]), append([]byte(nil), raw...)}, nil
}

type Receipt struct {
	DeliveryID string
	Event      string
	BodySHA256 string
}
type Replay string

const (
	New            Replay = "new"
	Duplicate      Replay = "duplicate"
	ReplayConflict Replay = "conflict"
)

// ClassifyReplay is only a comparison. The store must serialize lookup/insert,
// including lookup by event+body hash under a different delivery ID.
func ClassifyReplay(in Receipt, existing Receipt) Replay {
	if in.DeliveryID == existing.DeliveryID {
		if in.Event == existing.Event && in.BodySHA256 == existing.BodySHA256 {
			return Duplicate
		}
		return ReplayConflict
	}
	if in.Event == existing.Event && in.BodySHA256 == existing.BodySHA256 {
		return Duplicate
	}
	return New
}
