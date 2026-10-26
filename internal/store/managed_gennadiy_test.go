package store

import (
	"testing"

	"agent-board/internal/core"
)

const gennadiyTestSession = "ses_0123456789abcdefABCDEFghij"

func gennadiySetup(t *testing.T) *managedFixture {
	t.Helper()
	f := newFixture(t)
	a := f.actor(t, "human")
	s := settings()
	s.Provider = "gennadiy"
	p, err := f.s.ProfileCreate(testContext, f.coord, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: s})
	if err != nil {
		t.Fatal(err)
	}
	h, err := f.s.ManagedHostEnroll(testContext, a, core.ManagedHostEnrollInput{RequestID: f.id(), Name: "Local", HostSecret: managedTestSecret, Workspaces: []string{"gui"}, Providers: []string{"gennadiy"}, Controllers: []string{}})
	if err != nil {
		t.Fatal(err)
	}
	return &managedFixture{f, a, p, h}
}

func TestManagedGennadiyIsAThirdNativeProvider(t *testing.T) {
	f := newFixture(t)
	a := f.actor(t, "human")
	for _, providers := range [][]string{{"gennadiy"}, {"codex", "claude", "gennadiy"}} {
		if _, err := f.s.ManagedHostEnroll(testContext, a, core.ManagedHostEnrollInput{RequestID: f.id(), Name: "Local", HostSecret: managedTestSecret, Workspaces: []string{"gui"}, Providers: providers, Controllers: []string{}}); err != nil {
			t.Fatal(providers, err)
		}
	}
	for _, providers := range [][]string{{"opencode"}, {"gennadiy", "gennadiy"}, {"codex", "claude", "gennadiy", "shell"}} {
		if _, err := f.s.ManagedHostEnroll(testContext, a, core.ManagedHostEnrollInput{RequestID: f.id(), Name: "Local", HostSecret: managedTestSecret, Workspaces: []string{"gui"}, Providers: providers, Controllers: []string{}}); err == nil {
			t.Fatal("unsupported provider list accepted", providers)
		}
	}
	s := settings()
	s.Provider = "opencode"
	if _, err := f.s.ProfileCreate(testContext, f.coord, core.ProfileCreateInput{RequestID: f.id(), ProfileSettings: s}); err == nil {
		t.Fatal("unknown profile provider accepted")
	}
}

func TestManagedGennadiyStartRequiresMatchingProfileAndExactSession(t *testing.T) {
	m := gennadiySetup(t)
	fs := settings()
	fs.Provider = "openai"
	foreign, err := m.f.s.ProfileCreate(testContext, m.f.coord, core.ProfileCreateInput{RequestID: m.f.id(), ProfileSettings: fs})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "gennadiy", ProfileID: foreign.ID, ProfileVersion: foreign.Version, Prompt: "Mismatch"}); err == nil {
		t.Fatal("profile of another provider accepted for gennadiy")
	}
	if _, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "codex", ProfileID: m.p.ID, ProfileVersion: m.p.Version, Prompt: "Not enrolled"}); err == nil {
		t.Fatal("provider outside the local allowlist accepted")
	}
	r, err := m.f.s.ManagedStart(testContext, m.a, core.ManagedStartInput{RequestID: m.f.id(), HostID: m.h.ID, Workspace: "gui", Provider: "gennadiy", ProfileID: m.p.ID, ProfileVersion: m.p.Version, Prompt: "Read the README"})
	if err != nil || r.Run.Provider != "gennadiy" {
		t.Fatal("gennadiy start refused", err)
	}
	p := m.poll(t)
	m.report(t, p, "accepted")
	for _, bad := range []string{"ses_short", "ses_id with spaces", "opencode-session", "ses_" + gennadiyTestSession} {
		in := m.reportInput(p, "started")
		in.ProviderSessionID = bad
		if _, err = m.f.s.ManagedReport(testContext, m.a, in); err == nil {
			t.Fatalf("non-exact native session %q accepted", bad)
		}
	}
	in := m.reportInput(p, "started")
	in.ProviderSessionID = gennadiyTestSession
	if _, err = m.f.s.ManagedReport(testContext, m.a, in); err != nil {
		t.Fatal("OpenCode session ID refused", err)
	}
	in = m.reportInput(p, "completed")
	in.ProviderSessionID = gennadiyTestSession
	done, err := m.f.s.ManagedReport(testContext, m.a, in)
	if err != nil || done.Run.ProviderSessionID != gennadiyTestSession || done.Run.State != "completed" || done.Run.ID != r.Run.ID {
		t.Fatalf("exact OpenCode session not preserved: %+v %v", done.Run, err)
	}
}
