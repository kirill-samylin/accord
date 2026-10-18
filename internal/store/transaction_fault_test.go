package store

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"sync"
	"sync/atomic"
	"testing"

	"agent-board/internal/core"
	"modernc.org/sqlite"
)

var performanceCommitFault atomic.Int32
var performanceFaultDriverOnce sync.Once
var errPerformanceCommit = errors.New("injected indeterminate commit")

type performanceFaultDriver struct{ base driver.Driver }

func (d performanceFaultDriver) Open(name string) (driver.Conn, error) {
	conn, err := d.base.Open(name)
	if err != nil {
		return nil, err
	}
	return performanceFaultConn{Conn: conn}, nil
}

type performanceFaultConn struct{ driver.Conn }

func (c performanceFaultConn) Begin() (driver.Tx, error) {
	tx, err := c.Conn.Begin()
	if err != nil {
		return nil, err
	}
	return performanceFaultTx{Tx: tx}, nil
}

type performanceFaultTx struct{ driver.Tx }

func (t performanceFaultTx) Commit() error {
	mode := performanceCommitFault.Swap(0)
	if mode == 2 {
		if err := t.Tx.Rollback(); err != nil {
			return err
		}
		return errPerformanceCommit
	}
	if err := t.Tx.Commit(); err != nil {
		return err
	}
	if mode == 1 {
		return errPerformanceCommit
	}
	return nil
}

// The wrapper injects both possible outcomes after real SQLite's transaction
// boundary. No Store/mock return value substitutes for durable DB state.
func TestPerformanceIndeterminateCommitReloadsDurableReplay(t *testing.T) {
	performanceFaultDriverOnce.Do(func() { sql.Register("performance-commit-fault", performanceFaultDriver{base: &sqlite.Driver{}}) })
	for _, mode := range []int32{1, 2} {
		t.Run(map[int32]string{1: "durable", 2: "rolled_back"}[mode], func(t *testing.T) {
			f := newFixture(t)
			f.s.Close()
			dsn, err := prepareDatabase(f.path)
			if err != nil {
				t.Fatal(err)
			}
			db, err := sql.Open("performance-commit-fault", connectionDSN(dsn))
			if err != nil {
				t.Fatal(err)
			}
			db.SetMaxOpenConns(4)
			s := newStore(db, f.s.cfg, f.clock.read)
			defer s.Close()
			if _, err = s.Get(testContext, f.coord, core.GetInput{TaskKey: f.epic.Key}); err != nil {
				t.Fatal(err)
			}
			if s.cache == nil {
				t.Fatal("fixture did not warm committed cache")
			}
			input := core.CreateInput{RequestID: "uncertain", Type: "task", EpicKey: f.epic.Key, Role: "developer", Title: "uncertain result", Description: "synthetic"}
			performanceCommitFault.Store(mode)
			result, err := s.Create(testContext, f.coord, input)
			if !errors.Is(err, errPerformanceCommit) || result.Key != "" {
				t.Fatal("indeterminate commit returned success", result, err)
			}
			if s.cache != nil || s.writer != nil {
				t.Fatal("indeterminate commit retained cache generation")
			}
			var tasks, replays int
			if err = db.QueryRow(`SELECT count(*) FROM tasks WHERE json_extract(data,'$.title')='uncertain result'`).Scan(&tasks); err != nil {
				t.Fatal(err)
			}
			if err = db.QueryRow(`SELECT count(*) FROM idempotency WHERE key=?`, f.p.ID+"/"+f.coord.SessionID+"/uncertain").Scan(&replays); err != nil {
				t.Fatal(err)
			}
			expected := 0
			if mode == 1 {
				expected = 1
			}
			if tasks != expected || replays != expected {
				t.Fatal("partial durable outcome", tasks, replays)
			}
			retry, err := s.Create(testContext, f.coord, input)
			if err != nil || retry.Key == "" {
				t.Fatal(retry, err)
			}
			if err = db.QueryRow(`SELECT count(*) FROM tasks WHERE json_extract(data,'$.title')='uncertain result'`).Scan(&tasks); err != nil || tasks != 1 {
				t.Fatal("retry duplicated candidate", tasks, err)
			}
			replay, err := s.Create(testContext, f.coord, input)
			if err != nil || replay.Key != retry.Key {
				t.Fatal(replay, err)
			}
		})
	}
}
