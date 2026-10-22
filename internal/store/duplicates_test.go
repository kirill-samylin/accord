package store

import (
	"path/filepath"
	"testing"
	"time"

	"agent-board/internal/core"
)

// secondPerson opens a separate credential, as on another person's machine.
func (f *fixture) secondPerson(t *testing.T, owner, role string) core.Actor {
	t.Helper()
	token, err := f.s.CreateToken(testContext, core.TokenInput{Owner: owner, Machine: "pc-b", TTL: 24 * time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	p, err := f.s.Authenticate(testContext, token.Secret)
	if err != nil {
		t.Fatal(err)
	}
	v, err := f.s.Describe(testContext, p, core.OpenSession{RequestID: f.id(), Role: role, RunID: f.id(), Machine: p.Machine})
	if err != nil {
		t.Fatal(err)
	}
	return core.Actor{Principal: p, SessionID: v.SessionID}
}

func (f *fixture) titled(t *testing.T, a core.Actor, title string) core.Task {
	t.Helper()
	v, err := f.s.Create(testContext, a, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: title, Description: "scope", Role: "developer"})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCreateWarnsAboutAnotherPersonsSimilarTask(t *testing.T) {
	f := newFixture(t)
	first := f.titled(t, f.coord, "Добавить авторизацию через GitHub")
	if len(first.Warnings) != 0 || first.CreatedBy != "alice" {
		t.Fatalf("first task: %+v", first)
	}
	bob := f.secondPerson(t, "bob", "coordinator")
	second := f.titled(t, bob, "Авторизация через GitHub")
	if len(second.Warnings) != 1 || second.Warnings[0].Rule != "duplicate_check" || second.Warnings[0].Similar[0].Key != first.Key || second.Warnings[0].Similar[0].CreatedBy != "alice" {
		t.Fatalf("expected warning about %s: %+v", first.Key, second.Warnings)
	}
	other := f.titled(t, bob, "Экспорт задач в JSONL")
	if len(other.Warnings) != 0 {
		t.Fatalf("unrelated work warned: %+v", other.Warnings)
	}
	got, err := f.s.Get(testContext, f.coord, core.GetInput{TaskKey: second.Key})
	if err != nil || got.CreatedBy != "bob" || len(got.Warnings) != 0 {
		t.Fatalf("stored task must not persist warnings: %+v %v", got, err)
	}
}

func TestStrictModeRejectsUntilForcedWithReason(t *testing.T) {
	clock := &testClock{now: time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)}
	s, err := Open(filepath.Join(t.TempDir(), "b.db"), core.Config{SessionTTL: time.Hour, LeaseTTL: time.Minute, DuplicateMode: core.DuplicateStrict}, clock.read)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	f := &fixture{s: s, clock: clock}
	token, _ := s.CreateToken(testContext, core.TokenInput{Owner: "alice", Machine: "pc-a", TTL: time.Hour})
	f.p, _ = s.Authenticate(testContext, token.Secret)
	f.coord = f.actor(t, "coordinator")
	f.epic, _ = s.Create(testContext, f.coord, core.CreateInput{RequestID: "e", Type: "epic", Title: "Pilot"})
	original := f.titled(t, f.coord, "Экран настроек профиля")
	in := core.CreateInput{RequestID: "dup", Type: "task", EpicKey: f.epic.Key, Title: "Экран настроек профиля пользователя", Description: "x", Role: "developer"}
	_, err = s.Create(testContext, f.coord, in)
	requireCode(t, err, "DUPLICATE_SUSPECTED")
	in.RequestID, in.Force = "dup2", true
	_, err = s.Create(testContext, f.coord, in)
	requireCode(t, err, "VALIDATION")
	in.RequestID, in.ForceReason = "dup3", "Отдельный экран для администратора"
	forced, err := s.Create(testContext, f.coord, in)
	if err != nil || forced.DuplicateOverride == nil || forced.DuplicateOverride.Similar[0] != original.Key || len(forced.Warnings) != 1 {
		t.Fatalf("forced create: %+v %v", forced, err)
	}
	replay, err := s.Create(testContext, f.coord, in)
	if err != nil || replay.Key != forced.Key || len(replay.Warnings) != 1 {
		t.Fatalf("replay must return the stored response: %+v %v", replay, err)
	}
}

func TestDuplicateOfCancelsAndRestoreDetaches(t *testing.T) {
	f := newFixture(t)
	canonical := f.titled(t, f.coord, "Кнопка выхода")
	dup := f.titled(t, f.coord, "Кнопка выхода из аккаунта")
	marked, err := f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: dup.Key, To: canonical.Key, Type: "duplicate_of"})
	if err != nil || marked.State != "cancelled" || marked.DuplicateOf != canonical.Key {
		t.Fatalf("mark: %+v %v", marked, err)
	}
	_, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: canonical.Key, To: dup.Key, Type: "duplicate_of"})
	requireCode(t, err, "VALIDATION")
	restored, err := f.s.Transition(testContext, f.coord, core.TransitionInput{LeaseInput: core.LeaseInput{RequestID: f.id(), TaskKey: dup.Key, Reason: "это другое"}, State: "ready"})
	if err != nil || restored.DuplicateOf != "" || restored.State != "ready" {
		t.Fatalf("restore: %+v %v", restored, err)
	}
	page, err := f.s.Events(testContext, f.coord, core.EventsInput{Limit: 200, Types: []string{"task_duplicate"}})
	if err != nil || len(page.Events) != 1 || page.Events[0].Data.DuplicateOf != canonical.Key {
		t.Fatalf("event: %+v %v", page, err)
	}
}

func TestDuplicateNeverTakesALiveLease(t *testing.T) {
	f := newFixture(t)
	canonical := f.titled(t, f.coord, "Импорт JSONL")
	dup := f.titled(t, f.coord, "Импорт задач JSONL")
	dev := f.secondPerson(t, "bob", "developer")
	f.claim(t, dev, dup.Key)
	_, err := f.s.Duplicate(testContext, f.coord, core.DuplicateInput{RequestID: f.id(), TaskKey: dup.Key, CanonicalKey: canonical.Key})
	requireCode(t, err, "LEASE_HELD")
	developer := f.actor(t, "developer")
	stranger := f.titled(t, f.coord, "Совсем другое")
	_, err = f.s.Duplicate(testContext, developer, core.DuplicateInput{RequestID: f.id(), TaskKey: stranger.Key, CanonicalKey: canonical.Key, Version: 99})
	requireCode(t, err, "CONFLICT")
}

func TestClaimWarnsWhenAnotherPersonHoldsSimilarWork(t *testing.T) {
	f := newFixture(t)
	a := f.titled(t, f.coord, "Пагинация списка задач")
	bobCoord := f.secondPerson(t, "bob", "coordinator")
	b, err := f.s.Create(testContext, bobCoord, core.CreateInput{RequestID: f.id(), Type: "task", EpicKey: f.epic.Key, Title: "Пагинация в списке задач", Description: "x", Role: "developer", Force: true, ForceReason: "проверка"})
	if err != nil {
		t.Fatal(err)
	}
	bobDev := f.secondPerson(t, "bob", "developer")
	f.claim(t, bobDev, b.Key)
	aliceDev := f.actor(t, "developer")
	claim := f.claim(t, aliceDev, a.Key)
	if len(claim.Warnings) == 0 || claim.Warnings[0].Similar[0].HolderOwner != "bob" {
		t.Fatalf("expected warning about bob's live work: %+v", claim.Warnings)
	}
}

func TestTeamGroupsPeopleAgentsAndPairs(t *testing.T) {
	f := newFixture(t)
	a := f.titled(t, f.coord, "Тёмная тема панели")
	bob := f.secondPerson(t, "bob", "coordinator")
	f.titled(t, bob, "Темная тема для панели")
	dev := f.secondPerson(t, "bob", "developer")
	f.claim(t, dev, a.Key)
	team, err := f.s.ControlTeam(testContext, f.coord)
	if err != nil {
		t.Fatal(err)
	}
	if len(team.Members) != 2 || team.Members[0].Owner != "bob" || team.Members[0].InProgress != 1 || team.Members[0].Agents[0].Task.Key != a.Key {
		t.Fatalf("members: %+v", team.Members)
	}
	if team.DuplicatesTotal != 1 || team.Duplicates[0].Older.Key != a.Key || team.Duplicates[0].Newer.CreatedBy != "bob" {
		t.Fatalf("pairs: %+v", team.Duplicates)
	}
	// An explicit "related" link dismisses the pair.
	if _, err = f.s.Link(testContext, f.coord, core.LinkInput{RequestID: f.id(), From: a.Key, To: team.Duplicates[0].Newer.Key, Type: "related"}); err != nil {
		t.Fatal(err)
	}
	team, _ = f.s.ControlTeam(testContext, f.coord)
	if team.DuplicatesTotal != 0 {
		t.Fatalf("dismissed pair still listed: %+v", team.Duplicates)
	}
	page, err := f.s.Similar(testContext, f.coord, core.SimilarInput{Title: "тёмная тема"})
	if err != nil || len(page.Similar) != 2 || page.Mode != core.DuplicateWarn {
		t.Fatalf("similar: %+v %v", page, err)
	}
}
