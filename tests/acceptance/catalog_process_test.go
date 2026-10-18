package acceptance

import (
	"bytes"
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"agent-board/internal/core"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCatalogNotesProcessRestartPaginationResource(t *testing.T) {
	b := newBoard(t, "30s")
	c := newClient(t, b, "catalog-author")
	b.start()
	c.open(t, "developer", "catalog-author")
	coord := newClient(t, b, "catalog-coordinator")
	coord.open(t, "coordinator", "catalog-coordinator")
	task := coord.task(t, coord.epic(t).Key, "developer")
	create := c.prepare(t, payload{"request_id": "note-exact-replay", "action": "create", "scope_type": "role", "scope_id": "developer", "source_task_key": task.Key, "text": "Synthetic durable note"})
	first := c.raw("note", create)
	if first.err != nil {
		t.Fatal("CLI note create failed")
	}
	note := decode[core.Note](t, first.data)
	if note.SourceTaskKey != task.Key || note.AuthorSession != c.session || note.AuthorPrincipal == "" || note.CreatedAt.IsZero() {
		t.Fatal("note provenance incomplete")
	}
	for i := 0; i < 2; i++ {
		c.ok(t, "note", payload{"action": "create", "scope_type": "role", "scope_id": "developer", "source_task_key": task.Key, "text": fmt.Sprintf("Synthetic note %d", i)})
	}
	page := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer", "limit": 1}))
	if len(page.Notes) != 1 || page.Notes[0].Key != note.Key || !page.HasMore || page.NextCursor == "" {
		t.Fatal("notes first page incomplete")
	}
	b.stop()
	b.start()
	replay := c.raw("note", create)
	if replay.err != nil || !bytes.Equal(first.data, replay.data) {
		t.Fatal("restart changed exact persisted note replay")
	}
	// A later insert must not leak into the existing cursor pass.
	c.ok(t, "note", payload{"action": "create", "scope_type": "role", "scope_id": "developer", "source_task_key": task.Key, "text": "After high water"})
	seen := map[string]bool{note.Key: true}
	for page.HasMore {
		page = decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer", "limit": 1, "cursor": page.NextCursor}))
		for _, n := range page.Notes {
			if seen[n.Key] {
				t.Fatal("cursor duplicated a note")
			}
			seen[n.Key] = true
		}
	}
	if len(seen) != 3 {
		t.Fatal("restart cursor lost original notes or included later note")
	}
	worker := startMCP(t, c, b.url)
	mcpValue[core.SessionView](t, worker.call("describe", payload{"request_id": "notes-worker-open", "role": "developer", "run_id": "notes-worker", "machine": "catalog-author"}))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resource, err := worker.session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "accord://notes?scope_type=role&scope_id=developer&limit=200"})
	if err != nil || len(resource.Contents) != 1 {
		t.Fatal("stdio resource read failed")
	}
	resourcePage := decode[core.NotesPage](t, []byte(resource.Contents[0].Text))
	cliPage := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer", "limit": 200}))
	if !reflect.DeepEqual(resourcePage, cliPage) || len(cliPage.Notes) != 4 {
		t.Fatal("resource and CLI notes differ")
	}
	worker.stop()
	worker.assertProtocol(c.token)
	remove := c.prepare(t, payload{"request_id": "note-delete-replay", "action": "delete", "key": note.Key, "reason": "Synthetic cleanup"})
	deleted := c.raw("note", remove)
	if deleted.err != nil {
		t.Fatal("note delete failed")
	}
	removed := decode[core.Note](t, deleted.data)
	if removed.DeletedAt == nil || removed.DeleteReason != "Synthetic cleanup" || removed.AuthorPrincipal != note.AuthorPrincipal {
		t.Fatal("soft deletion lost provenance")
	}
	b.stop()
	b.start()
	if replay = c.raw("note", remove); replay.err != nil || !bytes.Equal(deleted.data, replay.data) {
		t.Fatal("delete replay changed after restart")
	}
	active := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer"}))
	all := decode[core.NotesPage](t, c.ok(t, "notes", payload{"scope_type": "role", "scope_id": "developer", "include_deleted": true}))
	if len(active.Notes) != 3 || len(all.Notes) != 4 || all.Notes[0].DeletedAt == nil {
		t.Fatal("soft deletion list filters did not survive restart")
	}
	t.Log("C-01 notes CLI CRUD/replay, cursor high water and stdio resource survived real server restarts")
}
