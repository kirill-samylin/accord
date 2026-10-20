package projects

import (
	"agent-board/internal/web"
	"database/sql"
	"net/http"
	"time"
)

func (r *Registry) login(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()
	if req.Method == "GET" || req.Method == "HEAD" {
		secret, e := random("", 32)
		if e != nil {
			fail(w, e)
			return
		}
		csrf, e := random("", 32)
		if e != nil {
			fail(w, e)
			return
		}
		expires := time.Now().Add(10 * time.Minute)
		_, e = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
			if e := cleanupCatalog(ctx, tx); e != nil {
				return false, e
			}
			var count int
			if e := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM preauth`).Scan(&count); e != nil {
				return false, e
			}
			if count >= 1024 {
				return false, denied()
			}
			_, e := tx.ExecContext(ctx, `INSERT INTO preauth VALUES(?,?,?)`, hash(secret), csrf, expires.UnixNano())
			return true, e
		})
		if e != nil {
			fail(w, e)
			return
		}
		r.cookie(w, r.preauthName(), secret, expires)
		r.options.Browser.RenderLogin(w, req, 200, csrf, "")
		return
	}
	if req.Method != "POST" {
		fail(w, invalid())
		return
	}
	fields, e := web.LoginFields(w, req)
	if e != nil {
		fail(w, invalid())
		return
	}
	cookies := req.CookiesNamed(r.preauthName())
	if len(cookies) != 1 || !validHex(cookies[0].Value, 64) {
		fail(w, denied())
		return
	}
	var csrf string
	var expiry int64
	e = r.db.QueryRowContext(ctx, `SELECT csrf,expires FROM preauth WHERE hash=?`, hash(cookies[0].Value)).Scan(&csrf, &expiry)
	if e != nil || expiry <= time.Now().UnixNano() || !equal(csrf, fields["csrf"]) {
		fail(w, denied())
		return
	}
	// The existing bounded login policy remains global to the broker and uses
	// only an explicitly trusted loopback proxy's overwritten client IP.
	undo, ok := r.options.Browser.ReserveBrokerLogin(req)
	if !ok {
		w.Header().Set("Retry-After", "60")
		write(w, 429, map[string]any{"ok": false, "error": map[string]string{"code": "RATE_LIMITED", "message": "Повторите вход через минуту"}})
		return
	}
	who, e := r.Authenticate(ctx, fields["token"])
	if e != nil {
		r.options.Browser.RenderLogin(w, req, 401, csrf, "Не удалось войти: проверьте токен")
		return
	}
	undo()
	secret, e := random("", 32)
	if e != nil {
		fail(w, e)
		return
	}
	nextCSRF, e := random("", 32)
	if e != nil {
		fail(w, e)
		return
	}
	until := time.Now().Add(8 * time.Hour)
	if who.Expires.Before(until) {
		until = who.Expires
	}
	if old, e := r.browser(req); e == nil {
		if e = r.endSession(ctx, old); e != nil {
			fail(w, e)
			return
		}
	}
	_, e = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
		if e := cleanupCatalog(ctx, tx); e != nil {
			return false, e
		}
		var expires int64
		if e := tx.QueryRowContext(ctx, `SELECT expires FROM preauth WHERE hash=?`, hash(cookies[0].Value)).Scan(&expires); e != nil || expires <= time.Now().UnixNano() {
			return false, denied()
		}
		if _, e := tx.ExecContext(ctx, `DELETE FROM preauth WHERE hash=?`, hash(cookies[0].Value)); e != nil {
			return false, e
		}
		_, e := tx.ExecContext(ctx, `INSERT INTO sessions VALUES(?,?,?,?)`, hash(secret), who.ID, nextCSRF, until.UnixNano())
		return true, e
	})
	if e != nil {
		fail(w, e)
		return
	}
	r.cookie(w, r.cookieName(), secret, until)
	r.cookie(w, r.preauthName(), "", time.Time{})
	// Explicit login may open the first accessible project; a B-only identity
	// never loads, describes or receives data from the default project.
	projects, e := r.List(ctx, who)
	if e != nil {
		fail(w, e)
		return
	}
	target := "/web/projects"
	if len(projects) > 0 {
		b := browserIdentity{Identity: who, Hash: hash(secret), CSRF: nextCSRF, Expires: until}
		target, e = r.switchProject(ctx, b, projects[0].ID, "")
		if e != nil {
			fail(w, e)
			return
		}
	}
	redirect(w, req, target)
}
