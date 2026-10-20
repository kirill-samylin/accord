package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/web"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIdenticalKeysAssetsAndForeignCursorsRemainProjectLocal(t *testing.T) {
	f := setup(t)
	p := f.create(t, f.who, "B")
	ids := []string{f.r.defaultID, p.ID}
	sessions := make([]core.SessionView, 2)
	keys := make([]string, 2)
	actors := make([]core.Actor, 2)
	for i, pid := range ids {
		sessions[i] = f.describe(t, pid, f.token.Secret, "session-"+pid, "human")
		ctx := accountContext(context.Background(), f.who)
		v, e := f.r.runtime(pid)
		if e != nil {
			t.Fatal(e)
		}
		principal, e := v.service.Authenticate(ctx, f.token.Secret)
		if e != nil {
			t.Fatal(e)
		}
		actors[i] = core.Actor{Principal: principal, SessionID: sessions[i].SessionID}
		for n := 0; n < 2; n++ {
			body, _ := json.Marshal(core.CreateInput{RequestID: pid + string(rune('0'+n)), Type: "epic", Title: pid + string(rune('0'+n))})
			out := f.native(t, "POST", "/api/v1/p/"+pid+"/tasks", f.token.Secret, sessions[i].SessionID, string(body))
			var task core.Task
			json.Unmarshal(out.Body.Bytes(), &task)
			if out.Code != 200 {
				t.Fatal(out.Body.String())
			}
			if n == 0 {
				keys[i] = task.Key
			}
		}
	}
	if keys[0] != keys[1] {
		t.Fatal("fixture did not create identical board-local keys")
	}
	own := f.native(t, "GET", "/api/v1/p/"+p.ID+"/tasks/"+keys[0], f.token.Secret, sessions[1].SessionID, "")
	if own.Code != 200 || !strings.Contains(own.Body.String(), p.ID+"0") || strings.Contains(own.Body.String(), f.r.defaultID+"0") {
		t.Fatalf("same-key task scope %d %s", own.Code, own.Body.String())
	}
	foreign := f.native(t, "GET", "/api/v1/p/"+p.ID+"/tasks/"+keys[0], f.token.Secret, sessions[0].SessionID, "")
	if foreign.Code != 401 {
		t.Fatalf("foreign session accepted %d", foreign.Code)
	}
	first := f.native(t, "GET", "/api/v1/p/"+ids[0]+"/epics?limit=1", f.token.Secret, sessions[0].SessionID, "")
	var page core.TasksPage
	json.Unmarshal(first.Body.Bytes(), &page)
	if first.Code != 200 || page.NextCursor == "" {
		t.Fatal("missing A cursor", first.Body.String())
	}
	foreign = f.native(t, "GET", "/api/v1/p/"+ids[1]+"/epics?limit=1&cursor="+page.NextCursor, f.token.Secret, sessions[1].SessionID, "")
	if foreign.Code != 400 {
		t.Fatalf("foreign task cursor accepted %d %s", foreign.Code, foreign.Body.String())
	}
	asset := strings.Repeat("a", 48)
	for i, pid := range ids {
		_, dir, e := f.r.projectPaths(pid)
		if e != nil {
			t.Fatal(e)
		}
		db, e := sql.Open("sqlite", filepath.Join(dir, "web.db"))
		if e != nil {
			t.Fatal(e)
		}
		for n, id := range []string{asset, strings.Repeat("b", 48)} {
			content := []byte(pid + " private design")
			sum := sha256.Sum256(content)
			if e = os.WriteFile(filepath.Join(dir, "blobs", id), content, 0600); e != nil {
				t.Fatal(e)
			}
			_, e = db.Exec(`INSERT INTO design_assets(asset_id,design_id,version,scope,scope_key,title,filename,media_type,size,sha256,uploader_principal,uploader_owner,created_at,status,request_id,metadata_hash,blob) VALUES(?,?,1,'project',?,?,?,'text/html',?,?,?,?,?,'ready',?,'fixture',?)`, id, id, pid, pid, "design.html", len(content), hex.EncodeToString(sum[:]), actors[i].Principal.ID, "operator", time.Now().UTC().Format(time.RFC3339Nano), pid+string(rune('0'+n)), id)
			if e != nil {
				t.Fatal(e)
			}
		}
		db.Close()
	}
	for i, pid := range ids {
		metadata := f.native(t, "GET", "/api/v1/p/"+pid+"/designs/"+asset, f.token.Secret, sessions[i].SessionID, "")
		var view web.DesignAsset
		json.Unmarshal(metadata.Body.Bytes(), &view)
		if metadata.Code != 200 || view.DownloadURL != "/api/v1/p/"+pid+"/designs/"+asset+"/download" {
			t.Fatalf("REST metadata scope %d %s", metadata.Code, metadata.Body.String())
		}
		req := httptest.NewRequest("GET", "http://127.0.0.1:8080"+view.DownloadURL, nil)
		req.Header.Set("Authorization", "Bearer "+f.token.Secret)
		req.Header.Set("X-Session", sessions[i].SessionID)
		out := &deadlineWriter{httptest.NewRecorder()}
		f.handler.ServeHTTP(out, req)
		if out.Code != 200 || out.Body.String() != pid+" private design" {
			t.Fatalf("same asset scope %d %s", out.Code, out.Body.String())
		}
	}
	a, _ := f.r.runtime(ids[0])
	b, _ := f.r.runtime(ids[1])
	ctx := accountContext(context.Background(), f.who)
	designs, e := a.browser.ListDesigns(ctx, actors[0], web.DesignFilter{Limit: 1})
	if e != nil || designs.NextCursor == "" {
		t.Fatal("design cursor", e)
	}
	if _, e = b.browser.ListDesigns(ctx, actors[1], web.DesignFilter{Limit: 1, Cursor: designs.NextCursor}); e == nil {
		t.Fatal("foreign design cursor accepted")
	}
}
