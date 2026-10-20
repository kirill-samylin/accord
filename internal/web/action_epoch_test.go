package web

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestWebActionEpochChildRowsPruneLogoutAndExpireAtomically(t *testing.T) {
	f := fixture(t)
	cookie := f.login(t, f.token(t, "epoch-cleanup"))
	b, e := f.s.loadBrowser(newBrowserRequest(cookie))
	if e != nil {
		t.Fatal(e)
	}
	assertNoOrphans := func() {
		t.Helper()
		var count int
		if e := f.s.db.QueryRow(`SELECT COUNT(*) FROM web_action_epochs e LEFT JOIN web_actions a ON a.id=e.id WHERE a.id IS NULL`).Scan(&count); e != nil || count != 0 {
			t.Fatalf("orphan epoch count %d %v", count, e)
		}
	}
	tx, e := f.s.db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	for n := 0; n < 2050; n++ {
		id := fmt.Sprintf("%064x", n)
		if _, e = tx.Exec(`INSERT INTO web_actions(id,session_hash,operation,created) VALUES(?,?,'create',?)`, id, b.Hash, n); e != nil {
			t.Fatal(e)
		}
		if _, e = tx.Exec(`INSERT INTO web_action_epochs VALUES(?,0)`, id); e != nil {
			t.Fatal(e)
		}
	}
	if e = tx.Commit(); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.issueAction(context.Background(), b, "create"); e != nil {
		t.Fatal(e)
	}
	assertNoOrphans()
	if e = f.s.deleteBrowser(context.Background(), b.Hash); e != nil {
		t.Fatal(e)
	}
	assertNoOrphans()
	cookie = f.login(t, f.token(t, "expired-cleanup"))
	b, e = f.s.loadBrowser(newBrowserRequest(cookie))
	if e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.issueAction(context.Background(), b, "create"); e != nil {
		t.Fatal(e)
	}
	if _, e = f.s.db.Exec(`UPDATE browser_sessions SET expires=? WHERE hash=?`, time.Now().Add(-time.Hour).UnixNano(), b.Hash); e != nil {
		t.Fatal(e)
	}
	f.s.cleanup()
	assertNoOrphans()
}
