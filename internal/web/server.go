package web

import (
	"agent-board/internal/core"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Server struct {
	board        Board
	options      Options
	db           *sql.DB
	mu           sync.Mutex
	designs      *designStore
	secure       bool
	cookieName   string
	stop         chan struct{}
	stopped      chan struct{}
	closeOnce    sync.Once
	loginMu      sync.Mutex
	loginBuckets map[string]*loginBucket
}
type browserSession struct {
	Hash, TokenID, CSRF string
	Expires             time.Time
	Session             core.SessionView
	Principal           core.Principal
	PendingDescribe     string
}

func (b *browserSession) actor() core.Actor {
	return core.Actor{Principal: b.Principal, SessionID: b.Session.SessionID}
}

func New(board Board, options Options) (*Server, error) {
	if board == nil {
		return nil, fmt.Errorf("board required")
	}
	u, err := url.Parse(options.Origin)
	if err != nil || u.User != nil || u.Host == "" || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || strings.Contains(u.Host, "\\") {
		return nil, fmt.Errorf("canonical web origin required")
	}
	secure := u.Scheme == "https"
	ip := net.ParseIP(u.Hostname())
	if !secure && !(u.Scheme == "http" && ip != nil && ip.IsLoopback()) {
		return nil, fmt.Errorf("HTTPS web origin required except explicit literal loopback development")
	}
	if u.Port() != "" {
		if _, err = net.LookupPort("tcp", u.Port()); err != nil {
			return nil, fmt.Errorf("invalid origin port")
		}
	}
	db, err := openWebDB(options.DataDir)
	if err != nil {
		return nil, err
	}
	s := &Server{board: board, options: options, db: db, secure: secure, cookieName: "__Host-accord_session", stop: make(chan struct{}), stopped: make(chan struct{}), loginBuckets: make(map[string]*loginBucket)}
	if !secure {
		s.cookieName = "accord_dev_session"
	}
	if err = s.initDesigns(); err != nil {
		db.Close()
		return nil, err
	}
	go s.cleanupLoop()
	return s, nil
}
func (s *Server) Close() error {
	s.closeOnce.Do(func() { close(s.stop); <-s.stopped })
	return s.db.Close()
}
func (s *Server) Ready(ctx context.Context) error { return s.db.PingContext(ctx) }
func randomID() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}
func digest(v string) string { b := sha256.Sum256([]byte(v)); return hex.EncodeToString(b[:]) }
func (s *Server) cookie(w http.ResponseWriter, name, value string, expires time.Time) {
	c := &http.Cookie{Name: name, Value: value, Path: "/", HttpOnly: true, Secure: s.secure, SameSite: http.SameSiteStrictMode, Expires: expires}
	if value == "" {
		c.MaxAge = -1
	}
	http.SetCookie(w, c)
}
func (s *Server) loadBrowser(r *http.Request) (*browserSession, error) {
	cookies := r.CookiesNamed(s.cookieName)
	if len(cookies) != 1 || len(cookies[0].Value) != 64 {
		return nil, core.Fail("unauthorized", "sign in required")
	}
	b := &browserSession{Hash: digest(cookies[0].Value)}
	var expiry int64
	var raw string
	err := s.db.QueryRowContext(r.Context(), `SELECT token_id,csrf,expires,board_session,pending_describe FROM browser_sessions WHERE hash=?`, b.Hash).Scan(&b.TokenID, &b.CSRF, &expiry, &raw, &b.PendingDescribe)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, core.Fail("unauthorized", "sign in required")
	}
	if err != nil {
		return nil, err
	}
	b.Expires = time.Unix(0, expiry)
	if !b.Expires.After(time.Now()) {
		return nil, core.Fail("unauthorized", "browser session expired")
	}
	if err = json.Unmarshal([]byte(raw), &b.Session); err != nil {
		return nil, err
	}
	p, until, err := s.board.AuthenticateID(r.Context(), b.TokenID)
	if err != nil {
		return nil, err
	}
	if !until.After(time.Now()) {
		return nil, core.Fail("unauthorized", "token expired")
	}
	b.Principal = p
	return b, nil
}
func (s *Server) saveSession(ctx context.Context, b *browserSession) error {
	raw, err := json.Marshal(b.Session)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE browser_sessions SET board_session=?,pending_describe=? WHERE hash=?`, string(raw), b.PendingDescribe, b.Hash)
	return err
}

// ensureSession is called only by an explicit continuation or a new create/claim.
// Its stable Describe input is durable before crossing into board.db.
func (s *Server) ensureSession(ctx context.Context, b *browserSession) error {
	if b.PendingDescribe == "" && b.Session.ExpiresAt.After(time.Now()) {
		return nil
	}
	if b.PendingDescribe == "" {
		id, err := randomID()
		if err != nil {
			return err
		}
		b.PendingDescribe = id
		if err = s.saveSession(ctx, b); err != nil {
			return err
		}
	}
	v, err := s.board.Describe(ctx, b.Principal, core.OpenSession{RequestID: b.PendingDescribe, Role: "human", RunID: b.PendingDescribe, Machine: b.Principal.Machine})
	if err != nil {
		return err
	}
	b.Session = v
	b.PendingDescribe = ""
	return s.saveSession(ctx, b)
}
