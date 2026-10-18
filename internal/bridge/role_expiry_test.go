package bridge

import (
	"agent-board/internal/core"
	"testing"
	"time"
)

func TestRoleLeaseBoundsWorkerDeadlineWithLongSession(t *testing.T) {
	started := time.Now()
	roleExpiry := started.Add(time.Second)
	b := &bridge{}
	v := core.SessionView{SessionID: "s", Role: "architect", RoleGeneration: 1, ExpiresAt: started.Add(time.Hour), RoleExpiresAt: &roleExpiry, SessionTTLSeconds: 3600, TaskTTLSeconds: 3600, RoleTTLSeconds: 1, HeartbeatEverySeconds: 1.0 / 3}
	if err := b.timing(v, started); err != nil {
		t.Fatal(err)
	}
	if b.sessionDeadline.After(roleExpiry) || b.interval > time.Second/3 || b.rpcTimeout > time.Second/3 {
		t.Fatal("role lease did not bound client safety horizon")
	}
	if b.live(started.Add(1500 * time.Millisecond)) {
		t.Fatal("expired role treated as live")
	}
}
