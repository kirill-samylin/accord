package projects

import (
	"agent-board/internal/core"
	"context"
	"crypto/subtle"
	"database/sql"
	"net/http"
	"strings"
	"time"
)

type browserIdentity struct {
	Identity   Identity
	Hash, CSRF string
	Expires    time.Time
}

func (r *Registry) cookieName() string {
	if strings.HasPrefix(r.options.Origin, "https:") {
		return "__Host-accord_session"
	}
	return "accord_dev_session"
}
func (r *Registry) preauthName() string {
	if strings.HasPrefix(r.options.Origin, "https:") {
		return "__Host-accord_preauth"
	}
	return "accord_dev_preauth"
}
func (r *Registry) cookie(w http.ResponseWriter, name, value string, expiry time.Time) {
	c := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: strings.HasPrefix(r.options.Origin, "https:"), SameSite: http.SameSiteStrictMode, Expires: expiry}
	if value == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}
func equal(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (r *Registry) browser(req *http.Request) (browserIdentity, error) {
	var out browserIdentity
	cookies := req.CookiesNamed(r.cookieName())
	if len(cookies) != 1 || !validHex(cookies[0].Value, 64) {
		return out, core.Fail("unauthorized", "sign in required")
	}
	out.Hash = hash(cookies[0].Value)
	var absent bool
	result, err := transaction(r, req.Context(), func(tx *sql.Tx) (browserIdentity, error) {
		var tomb int
		e := tx.QueryRowContext(req.Context(), `SELECT 1 FROM session_tombstones WHERE hash=?`, out.Hash).Scan(&tomb)
		if e == nil {
			return out, core.Fail("unauthorized", "session ended")
		}
		if !missing(e) {
			return out, e
		}
		var id string
		var expires int64
		e = tx.QueryRowContext(req.Context(), `SELECT identity_id,csrf,expires FROM sessions WHERE hash=?`, out.Hash).Scan(&id, &out.CSRF, &expires)
		if missing(e) {
			absent = true
			return out, nil
		}
		if e != nil {
			return out, e
		}
		out.Expires = time.Unix(0, expires)
		if !out.Expires.After(time.Now()) {
			return out, core.Fail("unauthorized", "session expired")
		}
		out.Identity, e = r.identity(req.Context(), tx, id)
		return out, e
	})
	if err != nil || !absent {
		return result, err
	}
	// Only a verified pre-existing default binding may be imported; tombstones
	// are checked again in the publication transaction to defeat logout races.
	oldHash, token, expiry, err := r.options.Browser.BrowserIdentity(req, r.options.Default)
	if err != nil {
		return out, core.Fail("unauthorized", "sign in required")
	}
	p, until, err := r.options.Default.AuthenticateID(req.Context(), token)
	if err != nil {
		return out, err
	}
	who, err := r.importLegacy(req.Context(), p, until)
	if err != nil {
		return out, err
	}
	return transaction(r, req.Context(), func(tx *sql.Tx) (browserIdentity, error) {
		var tomb int
		e := tx.QueryRowContext(req.Context(), `SELECT 1 FROM session_tombstones WHERE hash=?`, oldHash).Scan(&tomb)
		if e == nil {
			return out, core.Fail("unauthorized", "session ended")
		}
		if !missing(e) {
			return out, e
		}
		csrf, e := random("", 32)
		if e != nil {
			return out, e
		}
		_, e = tx.ExecContext(req.Context(), `INSERT INTO sessions VALUES(?,?,?,?) ON CONFLICT(hash) DO NOTHING`, oldHash, who.ID, csrf, expiry.UnixNano())
		if e != nil {
			return out, e
		}
		var id string
		var expires int64
		e = tx.QueryRowContext(req.Context(), `SELECT identity_id,csrf,expires FROM sessions WHERE hash=?`, oldHash).Scan(&id, &csrf, &expires)
		if e != nil {
			return out, e
		}
		current, e := r.identity(req.Context(), tx, id)
		return browserIdentity{Identity: current, Hash: oldHash, CSRF: csrf, Expires: time.Unix(0, expires)}, e
	})
}
func validHex(value string, length int) bool {
	if len(value) != length {
		return false
	}
	for _, c := range value {
		if c < '0' || c > '9' {
			if c < 'a' || c > 'f' {
				return false
			}
		}
	}
	return true
}
func (r *Registry) endSession(ctx context.Context, b browserIdentity) error {
	_, err := transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		_, e := tx.ExecContext(ctx, `INSERT INTO session_tombstones(hash,expires) VALUES(?,?) ON CONFLICT(hash) DO UPDATE SET expires=MAX(expires,excluded.expires)`, b.Hash, b.Expires.UnixNano())
		if e != nil {
			return false, e
		}
		if _, e = tx.ExecContext(ctx, `DELETE FROM sessions WHERE hash=?`, b.Hash); e != nil {
			return false, e
		}
		if _, e = tx.ExecContext(ctx, `DELETE FROM actions WHERE session_hash=?`, b.Hash); e != nil {
			return false, e
		}
		return true, audit(ctx, tx, "logout", b.Identity.ID, "")
	})
	if err != nil {
		return err
	}
	return r.options.Browser.ForgetProjectBinding(ctx, b.Hash)
}

// Carry the broker binding to every short project transaction. A request which
// passed the HTTP cookie check cannot start a later board commit after logout.
type globalBindingKey struct{}
type globalBinding struct{ Hash, Identity string }

func globalContext(ctx context.Context, b browserIdentity) context.Context {
	return context.WithValue(accountContext(ctx, b.Identity), globalBindingKey{}, globalBinding{Hash: b.Hash, Identity: b.Identity.ID})
}
func checkGlobal(ctx context.Context, tx *sql.Tx) error {
	binding, ok := ctx.Value(globalBindingKey{}).(globalBinding)
	if !ok {
		return nil
	}
	var identity string
	var expires int64
	err := tx.QueryRowContext(ctx, `SELECT identity_id,expires FROM sessions WHERE hash=? AND NOT EXISTS(SELECT 1 FROM session_tombstones WHERE hash=?)`, binding.Hash, binding.Hash).Scan(&identity, &expires)
	if err != nil || identity != binding.Identity || expires <= time.Now().UnixNano() {
		return core.Fail("unauthorized", "browser session ended")
	}
	return nil
}
