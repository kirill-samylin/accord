package web

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

// BrowserIdentity verifies a legacy binding against its original credential
// authority. It is only used to preserve an already signed-in default browser.
func (s *Server) BrowserIdentity(r *http.Request, authority interface {
	AuthenticateID(context.Context, string) (core.Principal, time.Time, error)
}) (string, string, time.Time, error) {
	cookies := r.CookiesNamed(s.cookieName)
	if len(cookies) != 1 || len(cookies[0].Value) != 64 {
		return "", "", time.Time{}, core.Fail("unauthorized", "sign in required")
	}
	hash := digest(cookies[0].Value)
	var token string
	var expires int64
	err := s.db.QueryRowContext(r.Context(), `SELECT token_id,expires FROM browser_sessions WHERE hash=?`, hash).Scan(&token, &expires)
	if err != nil {
		return "", "", time.Time{}, core.Fail("unauthorized", "sign in required")
	}
	_, until, err := authority.AuthenticateID(r.Context(), token)
	expiry := time.Unix(0, expires)
	if err != nil || !expiry.After(time.Now()) {
		return "", "", time.Time{}, core.Fail("unauthorized", "sign in required")
	}
	if until.Before(expiry) {
		expiry = until
	}
	return hash, token, expiry, nil
}

// ConfigureProject is called once before serving requests, never by a browser.
func (s *Server) ConfigureProject(board Board, options Options) {
	s.board = board
	s.options.ProjectID = options.ProjectID
	s.options.ProjectInfo = options.ProjectInfo
	s.options.AccessGuard = options.AccessGuard
	s.options.ActionEpoch = options.ActionEpoch
	s.options.SessionAllowed = options.SessionAllowed
}
func (s *Server) ProjectBinding(ctx context.Context, hash string) (core.SessionView, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT board_session FROM browser_sessions WHERE hash=?`, hash).Scan(&raw)
	var view core.SessionView
	if err == nil {
		err = json.Unmarshal([]byte(raw), &view)
	}
	return view, err
}

// OpenProject is an explicit login/switch operation. Recovery precedes Describe;
// GET requests never call it. Old epochs cannot revive cached web actions.
func (s *Server) OpenProject(ctx context.Context, hash string, p core.Principal, expires time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b := &browserSession{Hash: hash, Principal: p, TokenID: p.ID, Expires: expires}
	var raw string
	var existingToken string
	var storedExpiry int64
	err := s.db.QueryRowContext(ctx, `SELECT token_id,csrf,expires,board_session,pending_describe FROM browser_sessions WHERE hash=?`, hash).Scan(&existingToken, &b.CSRF, &storedExpiry, &raw, &b.PendingDescribe)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	fresh := errors.Is(err, sql.ErrNoRows)
	if !fresh {
		if existingToken != p.ID {
			return core.Fail("unauthorized", "browser identity mismatch")
		}
		if err = json.Unmarshal([]byte(raw), &b.Session); err != nil {
			return err
		}
		if err = s.recoverActions(ctx, b); err != nil {
			return err
		}
		if s.options.SessionAllowed != nil && b.Session.SessionID != "" && s.options.SessionAllowed(ctx, b.Session.SessionID) != nil {
			if err = s.deleteBrowser(ctx, hash); err != nil {
				return err
			}
			fresh = true
			b.Session = core.SessionView{}
			b.PendingDescribe = ""
		}
	}
	if fresh {
		b.CSRF, err = randomID()
		if err != nil {
			return err
		}
		rawBytes, _ := json.Marshal(b.Session)
		_, err = s.db.ExecContext(ctx, `INSERT INTO browser_sessions(hash,token_id,csrf,expires,board_session,pending_describe) VALUES(?,?,?,?,?,'')`, hash, p.ID, b.CSRF, expires.UnixNano(), string(rawBytes))
		if err != nil {
			return err
		}
	}
	return s.ensureSession(ctx, b)
}
func (s *Server) ForgetProjectBinding(ctx context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.deleteBrowser(ctx, hash)
}
func projectResponseContext(w http.ResponseWriter) context.Context {
	if out, ok := w.(*browserJSONWriter); ok {
		return out.request.Context()
	}
	return context.Background()
}

// LoginFields shares the bounded token/file parser with the registry broker.
func LoginFields(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	return loginFields(w, r)
}
func (s *Server) LoginResponse(w http.ResponseWriter, r *http.Request, status int, csrf, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if r.Method != http.MethodHead {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": status < 400, "page": map[string]any{"name": "login", "csrf": csrf, "error": message}})
	}
}
func (s *Server) ValidateLogout(r *http.Request) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, err := s.loadBrowser(r)
	if err != nil {
		return err
	}
	if !constantEqual(r.PostForm.Get("csrf"), b.CSRF) {
		return core.Fail("forbidden", "refresh logout form")
	}
	return s.checkAction(r.Context(), b, r.PostForm.Get("action_id"), "logout")
}

// ReserveBrokerLogin retains the bounded trusted-proxy policy for global login.
func (s *Server) ReserveBrokerLogin(r *http.Request) (func(), bool) { return s.reserveLogin(r) }
