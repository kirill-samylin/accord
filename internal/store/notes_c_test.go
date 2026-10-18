package store

import (
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestNotesProvenanceReplayDeleteQuotaAndSummaries(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	source := f.task(t, "developer")
	f.configure(t, func(c *config.Config) {
		c.Notes = config.Notes{MaxPerRole: 1, MaxBytesPerRole: 20, MaxPerEpic: 1, MaxBytesPerEpic: 20}
	})
	in := core.NoteInput{RequestID: f.id(), Action: "create", ScopeType: "role", ScopeID: "developer", SourceTaskKey: source.Key, Text: "remember"}
	n, err := f.s.Note(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := f.s.Note(testContext, d, in)
	if err != nil || !reflect.DeepEqual(n, replay) {
		t.Fatal("note replay changed")
	}
	if n.AuthorPrincipal != f.p.ID || n.AuthorSession != d.SessionID {
		t.Fatal("missing provenance")
	}
	in.RequestID = f.id()
	_, err = f.s.Note(testContext, d, in)
	requireCode(t, err, "LIMIT_EXCEEDED")
	if !strings.Contains(err.Error(), n.Key) {
		t.Fatal("quota omitted oldest key")
	}
	described, err := f.s.Describe(testContext, f.p, core.OpenSession{RequestID: f.id(), Role: "developer", RunID: f.id(), Machine: f.p.Machine})
	if err != nil || len(described.Notes) != 1 || described.Queue.Ready != 1 {
		t.Fatalf("describe summary %+v %v", described, err)
	}
	in.ScopeType = "epic"
	in.ScopeID = f.epic.Key
	in.RequestID = f.id()
	epicNote, err := f.s.Note(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	claim := f.claim(t, d, source.Key)
	if len(claim.EpicNotes) != 1 || claim.EpicNotes[0].Key != epicNote.Key || claim.Queue.InProgress != 1 {
		t.Fatal("next missing summary")
	}
	del := core.NoteInput{RequestID: f.id(), Action: "delete", Key: n.Key, Reason: "superseded"}
	_, err = f.s.Note(testContext, d, del)
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.ListNotes(testContext, f.coord, core.NotesInput{ScopeType: "role", ScopeID: "developer"})
	if err != nil || len(p.Notes) != 0 {
		t.Fatal("soft deleted visible")
	}
	p, err = f.s.ListNotes(testContext, f.coord, core.NotesInput{ScopeType: "role", ScopeID: "developer", IncludeDeleted: true})
	if err != nil || len(p.Notes) != 1 || p.Notes[0].DeletedAt == nil || p.Notes[0].AuthorSession != d.SessionID {
		t.Fatal("deletion lost provenance")
	}
	in.ScopeType = "role"
	in.ScopeID = "developer"
	in.RequestID = f.id()
	later, err := f.s.Note(testContext, d, in)
	if err != nil || numericKey(later.Key) <= numericKey(epicNote.Key) {
		t.Fatal("sequence reused")
	}
}
func TestNotesRejectForeignRoleAndPrincipalAndAtomicLimits(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	source := f.task(t, "developer")
	f.configure(t, func(c *config.Config) { c.Notes.MaxPerRole = 1 })
	in := core.NoteInput{Action: "create", ScopeType: "role", ScopeID: "architect", SourceTaskKey: source.Key, Text: "x", RequestID: f.id()}
	_, err := f.s.Note(testContext, d, in)
	requireCode(t, err, "FORBIDDEN_ROLE")
	other, err := Open(f.path, f.s.cfg, f.clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	in.ScopeID = "developer"
	inputs := []core.NoteInput{in, in}
	inputs[0].RequestID = "one"
	inputs[1].RequestID = "two"
	var wg sync.WaitGroup
	errors := make(chan error, 2)
	for i, s := range []*Store{f.s, other} {
		wg.Add(1)
		go func(s *Store, in core.NoteInput) {
			defer wg.Done()
			_, err := s.Note(testContext, d, in)
			errors <- err
		}(s, inputs[i])
	}
	wg.Wait()
	close(errors)
	success := 0
	for err := range errors {
		if err == nil {
			success++
		} else {
			requireCode(t, err, "LIMIT_EXCEEDED")
		}
	}
	if success != 1 {
		t.Fatalf("concurrent quota success %d", success)
	}
}

func TestNotesForeignCredentialByteLimitsAndAuditRollback(t *testing.T) {
	f := newFixture(t)
	d := f.actor(t, "developer")
	source := f.task(t, "developer")
	in := core.NoteInput{Action: "create", ScopeType: "role", ScopeID: "developer", SourceTaskKey: source.Key, Text: "experience", RequestID: f.id()}
	n, err := f.s.Note(testContext, d, in)
	if err != nil {
		t.Fatal(err)
	}
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: "bob", Machine: "pc-b", TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	sess, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: "other", Role: "developer", RunID: "run", Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	other := core.Actor{Principal: p, SessionID: sess.SessionID}
	_, err = f.s.Note(testContext, other, core.NoteInput{RequestID: "delete", Action: "delete", Key: n.Key, Reason: "foreign"})
	requireCode(t, err, "FORBIDDEN_ROLE")
	in.ScopeType = "epic"
	in.ScopeID = f.epic.Key
	in.RequestID = "foreign-source"
	_, err = f.s.Note(testContext, other, in)
	if err != nil {
		t.Fatalf("same role should access source task: %v", err)
	}
	in.ScopeType = "role"
	in.ScopeID = "developer"
	in.Text = strings.Repeat("я", 1025)
	in.RequestID = f.id()
	_, err = f.s.Note(testContext, d, in)
	requireCode(t, err, "VALIDATION")
	in.Text = "new note"
	in.RequestID = f.id()
	if _, err = f.s.db.Exec(`CREATE TRIGGER reject_note BEFORE INSERT ON events WHEN instr(NEW.data,'"type":"note_created"')>0 BEGIN SELECT RAISE(ABORT,'audit rejected'); END`); err != nil {
		t.Fatal(err)
	}
	if _, err = f.s.Note(testContext, d, in); err == nil {
		t.Fatal("expected rejected audit")
	}
	if _, err = f.s.db.Exec(`DROP TRIGGER reject_note`); err != nil {
		t.Fatal(err)
	}
	page, err := f.s.ListNotes(testContext, f.coord, core.NotesInput{ScopeType: "role", ScopeID: "developer"})
	if err != nil || len(page.Notes) != 1 {
		t.Fatal("failed note transaction partly persisted")
	}
	created, err := f.s.Note(testContext, d, in)
	if err != nil || created.Key != "N-3" {
		t.Fatalf("failed sequence/replay rollback %v %v", created, err)
	}
}
