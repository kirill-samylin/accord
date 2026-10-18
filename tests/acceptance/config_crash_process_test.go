package acceptance

import (
	"bytes"
	"os/exec"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestConfigAcknowledgedWritesSurviveSIGKILLProcess(t *testing.T) {
	b := newBoard(t, "30s")
	coord := newClient(t, b, "crash-coord")
	b.start()
	coord.open(t, "coordinator", "crash-coord")
	task := coord.task(t, coord.epic(t).Key, "developer")
	create := coord.prepare(t, payload{"request_id": "crash-note", "action": "create", "scope_type": "role", "scope_id": "coordinator", "source_task_key": task.Key, "text": "acknowledged note before process kill"})
	first := coord.raw("note", create)
	if first.err != nil {
		t.Fatal("prekill acknowledged note failed")
	}
	before := configView(t, b)
	history := command(nil, "config", "history", "--db", b.db)
	if history.err != nil {
		t.Fatal("prekill history unavailable")
	}
	if err := b.cmd.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-b.exited:
		if _, ok := err.(*exec.ExitError); !ok {
			t.Fatal("expected abrupt killed process exit")
		}
	case <-time.After(7 * time.Second):
		t.Fatal("killed owned process not reaped")
	}
	b.cmd = nil
	b.start()
	after := configView(t, b)
	historyAfter := command(nil, "config", "history", "--db", b.db)
	if before.SHA256 != after.SHA256 || before.ConfigVersion != after.ConfigVersion || historyAfter.err != nil || !bytes.Equal(history.data, historyAfter.data) {
		t.Fatal("acknowledged config/history lost on SIGKILL restart")
	}
	replay := coord.raw("note", create)
	if replay.err != nil || !bytes.Equal(first.data, replay.data) {
		t.Fatal("acknowledged note replay lost after abrupt process kill")
	}
	notes := decode[core.NotesPage](t, coord.ok(t, "notes", payload{"scope_type": "role", "scope_id": "coordinator"}))
	if len(notes.Notes) != 1 {
		t.Fatal("process kill lost or duplicated committed note")
	}
	t.Log("D056-14 actual SIGKILL after acknowledged commits; same DB restart retains config/history/note/exact replay; no in-flight commit or power-loss claim")
}
