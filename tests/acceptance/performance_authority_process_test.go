package acceptance

import (
	"bytes"
	"encoding/json"
	"testing"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestPerformanceReplayUsesCurrentSQLScopesAfterExpiryProcess(t *testing.T) {
	for _, restriction := range []string{"token-scopes", "config-human-allowlist"} {
		t.Run(restriction, func(t *testing.T) {
			b := configBoard(t, func(c *config.Config) {
				c.Leases.SessionTTL = "1s"
				c.Leases.RoleTTL = "1s"
				c.Leases.MaxSilent = "1s"
				c.Leases.SweepInterval = "1s"
				c.Auth.Local.Users = []string{"acceptance", "replay-human"}
			})
			c := mintRoles(t, b, "replay-human", "human,developer")
			b.startOptions()
			raw, err := json.Marshal(payload{"request_id": "same-described-request", "role": "human", "run_id": "replay-human", "machine": "replay-human"})
			if err != nil {
				t.Fatal(err)
			}
			status, body := configHTTP(t, c, "POST", "/api/v1/session", raw, true)
			if status != 200 {
				t.Fatal("REST initial describe failed")
			}
			view := decode[core.SessionView](t, body)
			status, replay := configHTTP(t, c, "POST", "/api/v1/session", raw, true)
			if status != 200 || !bytes.Equal(replay, body) {
				t.Fatal("initial exact describe replay failed")
			}
			b.waitExpired(view.ExpiresAt)
			// Expiry alone permits durable replay; it must not renew expired session.
			db := inspectDatabase(t, b.db)
			sessionBefore := sqlString(t, db, "SELECT data FROM sessions WHERE key='"+view.SessionID+"'")
			status, replay = configHTTP(t, c, "POST", "/api/v1/session", raw, true)
			if status != 200 || !bytes.Equal(replay, body) || sqlString(t, db, "SELECT data FROM sessions WHERE key='"+view.SessionID+"'") != sessionBefore {
				t.Fatal("expiry replay renewed session or lost durable response")
			}
			if restriction == "token-scopes" {
				result := command(nil, "token", "scopes", "set", "--db", b.db, "--id", c.tokenID, "--roles", "developer", "--expected-version", "1", "--actor", "qa072")
				if result.err != nil || decode[core.TokenInfo](t, result.data).ScopesVersion != 2 {
					t.Fatal("offline scope restriction failed")
				}
			} else {
				applyConfig(t, b, func(c *config.Config) { c.Auth.Local.Users = []string{"acceptance"} })
			}
			before := sqlState(t, db)
			pid := b.cmd.Process.Pid
			status, body = configHTTP(t, c, "POST", "/api/v1/session", raw, true)
			if status != 403 || domainCode(t, body) != "FORBIDDEN_ROLE" || sqlState(t, db) != before || pid != b.cmd.Process.Pid {
				t.Fatal("stale exact replay ignored current SQL restriction or changed durable state")
			}
			b.stop()
			b.startOptions()
			status, body = configHTTP(t, c, "POST", "/api/v1/session", raw, true)
			if status != 403 || domainCode(t, body) != "FORBIDDEN_ROLE" || sqlState(t, db) != before {
				t.Fatal("restart lost current authority restriction")
			}
			t.Log("P072-03 real REST exact replay after expiry checks SQL scopes/current config, no renewal or writes")
		})
	}
}
