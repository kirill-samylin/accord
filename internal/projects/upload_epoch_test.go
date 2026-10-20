package projects

import (
	"agent-board/internal/web"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

type deadlineWriter struct{ *httptest.ResponseRecorder }

func (*deadlineWriter) SetReadDeadline(time.Time) error  { return nil }
func (*deadlineWriter) SetWriteDeadline(time.Time) error { return nil }
func TestUploadPublicationRechecksOriginalSessionEpoch(t *testing.T) {
	for _, change := range []string{"revoke-regrant", "viewer"} {
		t.Run(change, func(t *testing.T) {
			f := setup(t)
			ctx := context.Background()
			p := f.create(t, f.who, "upload epoch")
			credential, who, e := f.r.CreateCredential(ctx, "uploader", "machine", time.Hour)
			if e != nil {
				t.Fatal(e)
			}
			if e = f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); e != nil {
				t.Fatal(e)
			}
			cookies := f.login(t, credential.Secret)
			page := f.request("GET", "/web/p/"+p.ID+"/designs", cookies, nil)
			var wire boardPage
			json.Unmarshal(page.Body.Bytes(), &wire)
			if wire.Page.Project == nil || wire.Page.Project.ID != p.ID {
				t.Fatalf("design page lost project %s", page.Body.String())
			}
			v, e := f.r.runtime(p.ID)
			if e != nil {
				t.Fatal(e)
			}
			writes := 0
			v.browser.ConfigureProject(v.service, web.Options{ProjectID: p.ID, ProjectInfo: v.service.CurrentProject, ActionEpoch: v.service.Epoch, SessionAllowed: v.service.SessionAllowed, AccessGuard: func(ctx context.Context, write bool, session string, call func() error) error {
				if write {
					writes++
					if writes == 2 {
						if change == "viewer" {
							if e := f.r.changeMember(ctx, f.who, p.ID, who.ID, "viewer", "set", 1); e != nil {
								t.Fatal(e)
							}
						} else {
							if e := f.r.changeMember(ctx, f.who, p.ID, who.ID, "", "revoke", 1); e != nil {
								t.Fatal(e)
							}
							if e := f.r.changeMember(ctx, f.who, p.ID, who.ID, "editor", "grant", 0); e != nil {
								t.Fatal(e)
							}
						}
					}
				}
				return v.service.Guard(ctx, write, session, call)
			}})
			var body bytes.Buffer
			form := multipart.NewWriter(&body)
			for _, pair := range [][2]string{{"csrf", wire.Page.CSRF}, {"action_id", wire.Page.Actions["upload"]}, {"title", "safe"}, {"scope", "project"}, {"scope_key", p.ID}} {
				if e = form.WriteField(pair[0], pair[1]); e != nil {
					t.Fatal(e)
				}
			}
			file, e := form.CreateFormFile("file", "design.html")
			if e != nil {
				t.Fatal(e)
			}
			file.Write([]byte("<!doctype html><title>safe</title>"))
			form.Close()
			req := httptest.NewRequest("POST", "http://127.0.0.1:8080/web/p/"+p.ID+"/designs", &body)
			req.Header.Set("Origin", "http://127.0.0.1:8080")
			req.Header.Set("Accept", "application/json")
			req.Header.Set("Content-Type", form.FormDataContentType())
			for _, c := range cookies {
				req.AddCookie(c)
			}
			out := &deadlineWriter{httptest.NewRecorder()}
			f.handler.ServeHTTP(out, req)
			if writes != 2 || out.Code < 400 {
				t.Fatalf("stale publish %d writes=%d %s", out.Code, writes, out.Body.String())
			}
			db, e := sql.Open("sqlite", filepath.Join(f.r.options.DataDir, "projects", p.ID, "web", "web.db"))
			if e != nil {
				t.Fatal(e)
			}
			defer db.Close()
			var count int
			if e = db.QueryRow(`SELECT COUNT(*) FROM design_assets WHERE status='ready'`).Scan(&count); e != nil || count != 0 {
				t.Fatalf("stale upload published %d %v", count, e)
			}
		})
	}
}
