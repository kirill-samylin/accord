package store

import (
	"agent-board/internal/core"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestManagedStopReservedCapacityAtFullMailboxAndHistory(t *testing.T) {
	for _, fullHistory := range []bool{false, true} {
		t.Run(fmt.Sprint(fullHistory), func(t *testing.T) {
			m := managedSetup(t)
			m.start(t)
			p := m.poll(t)
			m.report(t, p, "accepted")
			live := m.report(t, p, "started")
			if fullHistory {
				for i := 2; i <= 256; i++ {
					record := managedCommandRecord{Command: core.ManagedCommand{ID: fmt.Sprintf("history-%d", i), RunID: live.Run.ID, Sequence: int64(i), Generation: 1, Kind: "clarify", Text: "historical", Status: "applied", CreatedBy: m.a.Principal.ID}}
					raw, _ := json.Marshal(record)
					if _, err := m.f.s.db.Exec(`INSERT INTO managed_commands VALUES(?,?)`, record.Command.ID, string(raw)); err != nil {
						t.Fatal(err)
					}
				}
			} else {
				for i := 0; i < 16; i++ {
					q, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "clarify", Text: "queued"})
					if err != nil {
						t.Fatal(err)
					}
					live = q
				}
			}
			stop, err := m.f.s.ManagedCommand(testContext, m.a, core.ManagedCommandInput{RequestID: m.f.id(), RunID: live.Run.ID, Version: live.Run.Version, Kind: "stop"})
			if err != nil {
				t.Fatal("quota blocked emergency stop", err)
			}
			q := m.poll(t)
			if q.Command == nil || q.Command.ID != stop.Command.ID || q.Command.Kind != "stop" {
				t.Fatal("stop was not priority")
			}
		})
	}
}
func TestManagedListsAndEscapedHistoryStayWithinNativeResponseLimit(t *testing.T) {
	m := managedSetup(t)
	start := m.start(t)
	run := start.Run
	run.Prompt = strings.Repeat("<", 16384)
	run.Profile.Description = strings.Repeat("<", 4096)
	run.Profile.Instructions = strings.Repeat("<", 16384)
	run.Output = []string{strings.Repeat("<", 65536)}
	for i := 0; i < 128; i++ {
		r := run
		if i > 0 {
			r.ID = fmt.Sprintf("history-run-%d", i)
		}
		raw, _ := json.Marshal(r)
		if _, err := m.f.s.db.Exec(`INSERT INTO managed_runs VALUES(?,?) ON CONFLICT(key) DO UPDATE SET data=excluded.data`, r.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	for i := 2; i <= 256; i++ {
		c := managedCommandRecord{Command: core.ManagedCommand{ID: fmt.Sprintf("history-command-%d", i), RunID: run.ID, Sequence: int64(i), Kind: "clarify", Status: "applied", Text: strings.Repeat("<", 4096)}}
		raw, _ := json.Marshal(c)
		if _, err := m.f.s.db.Exec(`INSERT INTO managed_commands VALUES(?,?)`, c.Command.ID, string(raw)); err != nil {
			t.Fatal(err)
		}
	}
	page, err := m.f.s.ManagedList(testContext, m.a, core.ManagedListInput{})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(page)
	if len(raw) >= 1<<20 || len(page.Runs) != 128 || page.Runs[0].Prompt != "" || len(page.Runs[0].Output) != 0 || page.Runs[0].Profile.Description != "" {
		t.Fatal("list leaks unbounded detail", len(raw))
	}
	detail, err := m.f.s.ManagedRun(testContext, m.a, core.ManagedRunInput{RunID: run.ID})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ = json.Marshal(detail)
	if len(raw) >= 1<<20 || !detail.CommandsTruncated || len(detail.Commands) > 100 {
		t.Fatal("escaped history exceeds native cap", len(raw), len(detail.Commands))
	}
}
