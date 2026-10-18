package acceptance

import (
	"bytes"
	"fmt"
	"testing"

	"agent-board/internal/core"
)

func TestPerformanceCreateReopenDurableRejectedAndReplayProcess(t *testing.T) {
	for _, kind := range []string{"developer", "verifier-same", "verifier-cross"} {
		t.Run(kind, func(t *testing.T) {
			b := newBoard(t, "30s")
			coord := newClient(t, b, "reopen-coord")
			dev := newClient(t, b, "reopen-dev")
			worker := newClient(t, b, "reopen-worker")
			b.start()
			coord.open(t, "coordinator", "reopen-coord")
			dev.open(t, "developer", "reopen-dev")
			epic := coord.epic(t)
			in := payload{"request_id": "reopen-durable", "type": "task", "epic_key": epic.Key, "role": "developer", "title": "Reopened work", "description": "independent process regression"}
			if kind != "developer" {
				target := coord.task(t, epic.Key, "developer")
				claim := dev.claim(t, target.Key)
				dev.ok(t, "transition", payload{"task_key": target.Key, "generation": claim.Lease.Generation, "state": "done", "evidence": payload{"commit": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", "run_summary": "synthetic"}})
				in["role"] = "reviewer"
				in["target_key"] = target.Key
				in["target_commit"] = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
			}
			coord.ok(t, "transition", payload{"task_key": epic.Key, "state": "done"})
			closed := []core.Task{decode[core.Task](t, coord.ok(t, "get", payload{"task_key": epic.Key}))}
			if kind == "verifier-cross" {
				parent := coord.epic(t)
				coord.ok(t, "transition", payload{"task_key": parent.Key, "state": "done"})
				closed = append(closed, decode[core.Task](t, coord.ok(t, "get", payload{"task_key": parent.Key})))
				in["epic_key"] = parent.Key
			}
			db := inspectDatabase(t, b.db)
			for _, field := range []string{"areas", "complexity"} {
				if field == "areas" {
					in[field] = []string{"../outside"}
				} else {
					in[field] = "invalid"
				}
				before := sqlState(t, db)
				coord.fails(t, "create", "VALIDATION", in)
				if sqlState(t, db) != before {
					t.Fatal("rejected create partially changed tables")
				}
				for _, old := range closed {
					got := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": old.Key}))
					if got.State != "done" || got.Version != old.Version {
						t.Fatal("rejected create polluted live epic cache")
					}
				}
				b.stop()
				b.start()
				if sqlState(t, db) != before {
					t.Fatal("rejected create changed state across restart")
				}
				delete(in, field)
			}
			raw := coord.prepare(t, in)
			created := coord.raw("create", raw)
			if created.err != nil {
				t.Fatal("corrected request_id reserved by rejected create")
			}
			child := decode[core.Task](t, created.data)
			for _, old := range closed {
				got := decode[core.Task](t, coord.ok(t, "get", payload{"task_key": old.Key}))
				persisted := decode[core.Task](t, []byte(sqlString(t, db, "SELECT data FROM tasks WHERE key='"+old.Key+"'")))
				if got.State != "open" || persisted.State != "open" || got.Version != old.Version+1 || persisted.Version != old.Version+1 || sqlString(t, db, "SELECT count(*) FROM events WHERE json_extract(data,'$.type')='epic_reopened' AND json_extract(data,'$.task_key')='"+old.Key+"'") != "1" {
					t.Fatal("reopen cache/SQL/version/audit mismatch")
				}
			}
			before := sqlState(t, db)
			for _, restart := range []bool{false, true} {
				if restart {
					b.stop()
					b.start()
				}
				replay := coord.raw("create", raw)
				if replay.err != nil || !bytes.Equal(replay.data, created.data) || sqlState(t, db) != before {
					t.Fatal("exact reopen replay/restart changed SQL/session/audit")
				}
			}
			worker.open(t, fmt.Sprint(in["role"]), "reopen-worker")
			claim := worker.claim(t, child.Key)
			if claim.Task == nil || claim.Task.Key != child.Key {
				t.Fatal("reopened child unavailable after restart")
			}
			t.Log("P072-04 rejected create/cache/SQL/restart then corrected request, one reopen and exact replay")
		})
	}
}
