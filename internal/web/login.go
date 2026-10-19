package web

import (
	"agent-board/internal/core"
	"context"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net"
	"net/http"
	"strings"
	"time"
)

type loginBucket struct {
	Start time.Time
	Count int
}

func (s *Server) loginKey(r *http.Request) string {
	if s.options.TrustedProxy {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		ip := net.ParseIP(host)
		if err == nil && ip != nil && ip.IsLoopback() {
			v := r.Header.Values("X-Accord-Client-IP")
			if len(v) == 1 {
				if client := net.ParseIP(v[0]); client != nil {
					return client.String()
				}
			}
		}
	}
	return "global"
}

// An attempt is reserved before authentication, so concurrent failures cannot
// bypass the bound. The default global bucket trusts no forwarded headers.
func (s *Server) reserveLogin(r *http.Request) (func(), bool) {
	s.loginMu.Lock()
	defer s.loginMu.Unlock()
	now := time.Now()
	key := s.loginKey(r)
	for k, b := range s.loginBuckets {
		if now.Sub(b.Start) >= time.Minute {
			delete(s.loginBuckets, k)
		}
	}
	b := s.loginBuckets[key]
	if b == nil {
		if len(s.loginBuckets) >= 1024 {
			key = "overflow"
			b = s.loginBuckets[key]
		}
		if b == nil {
			b = &loginBucket{Start: now}
			s.loginBuckets[key] = b
		}
	}
	if b.Count >= 5 {
		return nil, false
	}
	b.Count++
	return func() {
		s.loginMu.Lock()
		defer s.loginMu.Unlock()
		if s.loginBuckets[key] == b && b.Count > 0 {
			b.Count--
		}
	}, true
}
func (s *Server) preauthName() string {
	if s.secure {
		return "__Host-accord_preauth"
	}
	return "accord_dev_preauth"
}
func (s *Server) loginPage(w http.ResponseWriter, r *http.Request, message string, status int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	secret, err := randomID()
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	csrf, err := randomID()
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	_, err = s.db.ExecContext(r.Context(), `DELETE FROM preauth WHERE expires<=?`, time.Now().UnixNano())
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	var count int
	if err = s.db.QueryRowContext(r.Context(), `SELECT COUNT(*) FROM preauth`).Scan(&count); err != nil {
		s.fail(w, r, nil, err)
		return
	}
	if count >= 1024 {
		http.Error(w, "Вход временно недоступен", 503)
		return
	}
	expiry := time.Now().Add(10 * time.Minute)
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO preauth(hash,csrf,expires) VALUES(?,?,?)`, digest(secret), csrf, expiry.UnixNano())
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	s.cookie(w, s.preauthName(), secret, expiry)
	s.render(w, status, "login", Page{Title: "Вход в Accord", Error: message, CSRF: csrf})
}
func constantEqual(a, b string) bool {
	return len(a) == len(b) && subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	fields, err := loginFields(w, r)
	if err != nil {
		s.render(w, 400, "login", Page{Title: "Вход в Accord", Error: "Нужен один токен или файл токена до 4096 байт"})
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	cookies := r.CookiesNamed(s.preauthName())
	var csrf string
	var expires int64
	if len(cookies) != 1 {
		http.Error(w, "Обновите форму входа", 403)
		return
	}
	err = s.db.QueryRowContext(r.Context(), `SELECT csrf,expires FROM preauth WHERE hash=?`, digest(cookies[0].Value)).Scan(&csrf, &expires)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			s.fail(w, r, nil, err)
		} else {
			http.Error(w, "Обновите форму входа", 403)
		}
		return
	}
	if expires <= time.Now().UnixNano() || !constantEqual(fields["csrf"], csrf) {
		http.Error(w, "Обновите форму входа", 403)
		return
	}
	undo, ok := s.reserveLogin(r)
	if !ok {
		w.Header().Set("Retry-After", "60")
		s.render(w, 429, "login", Page{Title: "Вход в Accord", Error: "Слишком много попыток. Повторите через минуту", CSRF: csrf})
		return
	}
	p, err := s.board.Authenticate(r.Context(), fields["token"])
	if err != nil {
		s.render(w, errorStatus(err), "login", Page{Title: "Вход в Accord", Error: publicError(err), CSRF: csrf})
		return
	}
	p, until, err := s.board.AuthenticateID(r.Context(), p.ID)
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	undo()
	secret, err := randomID()
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	newCSRF, err := randomID()
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	expiry := time.Now().Add(8 * time.Hour)
	if until.Before(expiry) {
		expiry = until
	}
	b := &browserSession{Hash: digest(secret), TokenID: p.ID, CSRF: newCSRF, Principal: p, Expires: expiry}
	raw, _ := json.Marshal(b.Session)
	_, err = s.db.ExecContext(r.Context(), `INSERT INTO browser_sessions(hash,token_id,csrf,expires,board_session,pending_describe) VALUES(?,?,?,?,?,'')`, b.Hash, b.TokenID, b.CSRF, b.Expires.UnixNano(), string(raw))
	if err == nil {
		err = s.ensureSession(r.Context(), b)
	}
	if err != nil {
		s.fail(w, r, nil, err)
		return
	}
	if _, err = s.db.ExecContext(r.Context(), `DELETE FROM preauth WHERE hash=?`, digest(cookies[0].Value)); err != nil {
		s.fail(w, r, nil, err)
		return
	}
	// Successful reauthentication rotates and invalidates any previous browser ID.
	if old := r.CookiesNamed(s.cookieName); len(old) == 1 {
		if err = s.deleteBrowser(r.Context(), digest(old[0].Value)); err != nil {
			s.fail(w, r, nil, err)
			return
		}
	}
	s.cookie(w, s.cookieName, secret, expiry)
	s.cookie(w, s.preauthName(), "", time.Time{})
	http.Redirect(w, r, "/web/", http.StatusSeeOther)
}
func loginFields(w http.ResponseWriter, r *http.Request) (map[string]string, error) {
	r.Body = http.MaxBytesReader(w, r.Body, 16*1024)
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil {
		return nil, err
	}
	fields := map[string]string{}
	if media == "application/x-www-form-urlencoded" {
		if err = r.ParseForm(); err != nil {
			return nil, err
		}
		for k, v := range r.PostForm {
			if len(v) != 1 || k != "token" && k != "csrf" {
				return nil, core.Fail("validation", "invalid login fields")
			}
			fields[k] = v[0]
		}
	}
	if media == "multipart/form-data" {
		reader, e := r.MultipartReader()
		if e != nil {
			return nil, e
		}
		for n := 0; ; n++ {
			part, e := reader.NextPart()
			if e == io.EOF {
				break
			}
			if e != nil || n >= 3 {
				return nil, core.Fail("validation", "invalid login multipart")
			}
			name := part.FormName()
			if name != "token" && name != "token_file" && name != "csrf" {
				return nil, core.Fail("validation", "invalid field")
			}
			if _, ok := fields[name]; ok {
				return nil, core.Fail("validation", "duplicate field")
			}
			data, e := io.ReadAll(io.LimitReader(part, 4097))
			part.Close()
			if e != nil || len(data) > 4096 {
				return nil, core.Fail("validation", "token file too large")
			}
			fields[name] = string(data)
		}
	} else if media != "application/x-www-form-urlencoded" {
		return nil, core.Fail("validation", "unsupported login form")
	}
	a, b := strings.TrimSpace(fields["token"]), strings.TrimSpace(fields["token_file"])
	if a != "" && b != "" || a == "" && b == "" || len(a) > 4096 || len(b) > 4096 {
		return nil, core.Fail("validation", "one token source required")
	}
	if b != "" {
		a = b
	}
	fields["token"] = a
	delete(fields, "token_file")
	return fields, nil
}
func (s *Server) deleteBrowser(ctx context.Context, hash string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `DELETE FROM web_actions WHERE session_hash=?`, hash); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM browser_sessions WHERE hash=?`, hash); err != nil {
		return err
	}
	return tx.Commit()
}
