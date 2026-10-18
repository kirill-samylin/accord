package acceptance

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

func TestConfigOfflineCASAuditAndLiveReloadProcess(t *testing.T) {
	b := configBoard(t, func(c *config.Config) { c.Notes.MaxPerRole = 1 })
	c := newClient(t, b, "config-worker")
	path, v := configFile(t, b, func(c *config.Config) { c.Notes.MaxPerRole = 2 })
	valid := command(nil, "config", "validate", path)
	if valid.err != nil || !decode[config.Result](t, valid.data).Valid {
		t.Fatal("offline validate failed")
	}
	before := eventCountProcess(t, b, "config_applied")
	applied := applyFile(b, path, v)
	if applied.err != nil {
		t.Fatalf("offline apply: %s", applied.stderr)
	}
	next := decode[core.ConfigView](t, applied.data)
	if next.ConfigVersion != v+1 {
		t.Fatal("apply did not advance CAS")
	}
	noop := applyFile(b, path, v+1)
	if noop.err != nil || decode[core.ConfigView](t, noop.data).ConfigVersion != v+1 {
		t.Fatal("same-hash no-op version drift")
	}
	localFails(t, applyFile(b, path, v), "CONFIG_CONFLICT")
	if eventCountProcess(t, b, "config_applied") != before+2 {
		t.Fatal("no-op/rejection audit incorrect")
	}
	b.start()
	c.open(t, "coordinator", "config-worker")
	source := c.task(t, c.epic(t).Key, "developer")
	note := func(text string) payload {
		return payload{"action": "create", "scope_type": "role", "scope_id": "coordinator", "source_task_key": source.Key, "text": text}
	}
	c.ok(t, "note", note("first"))
	c.ok(t, "note", note("second"))
	c.fails(t, "note", "LIMIT_EXCEEDED", note("third"))
	worker := newClient(t, b, "config-leased-worker")
	worker.open(t, "developer", "config-leased-worker")
	claimed := worker.claim(t, source.Key)
	db := inspectDatabase(t, b.db)
	leaseBefore := sqlString(t, db, "SELECT data FROM tasks WHERE key='"+source.Key+"'")
	pid := b.cmd.Process.Pid
	live := applyConfig(t, b, func(c *config.Config) { c.Notes.MaxPerRole = 3 })
	created := decode[core.Note](t, c.ok(t, "note", note("third")))
	if created.Key == "" || configView(t, b).ConfigVersion != live.ConfigVersion || b.cmd.Process.Pid != pid {
		t.Fatal("live server did not load other-process apply")
	}
	// Rules are conservatively protected during live work. Expire the short session
	// before changing rules, while keeping the actual server PID unchanged.
	bad, version := configFile(t, b, func(c *config.Config) { c.Leases.TaskTTL = "40s" })
	rejected := applyFile(b, bad, version)
	localFails(t, rejected, "CONFIG_IN_USE")
	if !bytes.Contains(rejected.stderr, []byte("task:"+source.Key)) || !bytes.Contains(rejected.stderr, []byte("session:"+worker.session)) || sqlString(t, db, "SELECT data FROM tasks WHERE key='"+source.Key+"'") != leaseBefore {
		t.Fatal("CONFIG_IN_USE omitted concrete keys or changed live lease")
	}
	worker.ok(t, "heartbeat", payload{"task_key": source.Key, "generation": claimed.Lease.Generation})
	c.ok(t, "heartbeat", payload{})
	status, body := configHTTP(t, c, "GET", "/api/v1/config", nil, true)
	if status != 200 || decode[core.ConfigView](t, body).ConfigVersion != live.ConfigVersion || bytes.Contains(body, []byte("raw_yaml")) {
		t.Fatal("authenticated canonical config view incorrect")
	}
	status, _ = configHTTP(t, c, "GET", "/api/v1/config", nil, false)
	if status != 401 {
		t.Fatal("unauthenticated config read permitted")
	}
	status, body = configHTTP(t, c, "GET", "/api/v1/config/history?after_version=0&limit=1", nil, true)
	history := decode[core.ConfigHistory](t, body)
	if status != 200 || len(history.Revisions) != 1 || !history.HasMore || bytes.Contains(body, []byte("raw_yaml")) {
		t.Fatal("numeric config history pagination incorrect")
	}
	for _, route := range []string{"/api/v1/config/apply", "/api/v1/token/scopes", "/api/v1/token/create"} {
		status, _ = configHTTP(t, c, "POST", route, []byte(`{}`), true)
		if status != 404 && status != 405 {
			t.Fatalf("remote admin route %s status=%d", route, status)
		}
	}
	t.Log("D056-01 offline validate/CAS/no-op audit, other-process live note quota reload, protected TTL, read auth/history and absent remote admin routes")
}

func TestConfigConcurrentApplyProcessAndAuditRollback(t *testing.T) {
	b := configBoard(t, func(c *config.Config) { c.Notes.MaxPerRole = 2 })
	a, v := configFile(t, b, func(c *config.Config) { c.Notes.MaxPerRole = 3 })
	z, _ := configFile(t, b, func(c *config.Config) { c.Notes.MaxPerRole = 4 })
	auditBefore := eventCountProcess(t, b, "config_applied")
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	for _, f := range []string{a, z} {
		wg.Add(1)
		go func(path string) { defer wg.Done(); <-start; results <- applyFile(b, path, v) }(f)
	}
	close(start)
	wg.Wait()
	close(results)
	winners := 0
	winnerSHA := ""
	badCode := false
	var observed []result
	for r := range results {
		observed = append(observed, r)
		if r.err == nil {
			winners++
			winnerSHA = decode[core.ConfigView](t, r.data).SHA256
		} else if !bytes.Contains(r.stderr, []byte("CONFIG_CONFLICT")) {
			badCode = true
		}
	}
	current := configView(t, b)
	inspect := inspectDatabase(t, b.db)
	histories := sqlString(t, inspect, "SELECT count(*) FROM config_history")
	auditAfter := eventCountProcess(t, b, "config_applied")
	recordCAS(t, b, []string{a, z}, v, observed, current, histories, auditBefore, auditAfter, winners)
	if winners != 1 || current.ConfigVersion != v+1 || current.SHA256 != winnerSHA || histories != fmt.Sprint(v+1) || auditAfter != auditBefore+1 {
		t.Fatal("concurrent apply winner/version/hash/history/audit not atomic")
	}
	t.Logf("CAS poststate: winners=%d active=%d history=%s config_applied_delta=%d", winners, current.ConfigVersion, histories, auditAfter-auditBefore)
	if badCode {
		for _, r := range observed {
			if r.err != nil {
				t.Errorf("CAS expected CONFIG_CONFLICT; exact stderr=%q", r.stderr)
			}
		}
		return
	}
	before := configView(t, b)
	audits := eventCountProcess(t, b, "config_applied")
	db := inspectDatabase(t, b.db)
	if _, err := db.Exec(`CREATE TRIGGER qa056_reject_config_audit BEFORE INSERT ON events WHEN json_extract(NEW.data,'$.type')='config_applied' BEGIN SELECT RAISE(ABORT,'qa056 audit unavailable'); END`); err != nil {
		t.Fatal(err)
	}
	f, n := configFile(t, b, func(c *config.Config) { c.Notes.MaxPerRole = 5 })
	failed := applyFile(b, f, n)
	if failed.err == nil {
		t.Fatal("injected SQLite audit failure was ignored")
	}
	after := configView(t, b)
	if before.ConfigVersion != after.ConfigVersion || before.SHA256 != after.SHA256 || eventCountProcess(t, b, "config_applied") != audits || sqlString(t, db, "SELECT count(*) FROM config_history") != fmt.Sprint(before.ConfigVersion) {
		t.Fatal("failed audit partially wrote config/history")
	}
	if _, err := db.Exec("DROP TRIGGER qa056_reject_config_audit"); err != nil {
		t.Fatal(err)
	}
	raw, _ := os.ReadFile(f)
	raw = append(raw, []byte("\nunknown: true")...)
	if err := os.WriteFile(f, raw, 0600); err != nil {
		t.Fatal(err)
	}
	if applyFile(b, f, n).err == nil || configView(t, b).SHA256 != before.SHA256 {
		t.Fatal("invalid config partially applied")
	}
	t.Log("D056-02 two subprocess CAS one winner; isolated SQLite audit fault and invalid candidate have zero partial config/history/audit writes")
}

func TestConfigStartupBootstrapAndPersistedInputsProcess(t *testing.T) {
	empty := newBoard(t, "30s")
	localFails(t, command(nil, "serve", "--db", empty.db, "--listen", "127.0.0.1:0"), "CONFIG_BOOTSTRAP_REQUIRED")
	b := configBoard(t, func(c *config.Config) { c.Project.Name = "qa056-persisted" })
	c := newClient(t, b, "startup-worker")
	localFails(t, command(nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0", "--task-ttl", "15m"), "CONFIG_CONFLICT")
	localFails(t, command(nil, "serve", "--db", b.db, "--listen", "127.0.0.1:0", "--project", "other"), "CONFIG_CONFLICT")
	// This start omits TTL/project flags; implicit CLI defaults must not conflict.
	implicit := startExistingImplicit(t, b)
	c.open(t, "developer", "startup-worker")
	if configView(t, b).ConfigVersion != 2 || implicit != b.cmd.Process.Pid {
		t.Fatal("implicit defaults changed persisted config")
	}
	if !strings.Contains(string(configView(t, b).Config), "qa056-persisted") {
		t.Fatal("persisted project overwritten")
	}
	b.stop()
	startExistingImplicit(t, b)
	if configView(t, b).ConfigVersion != 2 {
		t.Fatal("restart changed persisted version")
	}
	t.Log("D056-03 missing bootstrap rejected, explicit conflicts rejected, implicit defaults and restart preserve persisted project/TTLs")
}
