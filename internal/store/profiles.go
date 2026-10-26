package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"agent-board/internal/core"
)

var profileID = regexp.MustCompile(`^ap_[0-9a-f]{32}$`)

func validateProfile(v core.ProfileSettings) error {
	if !utf8.ValidString(v.Name+v.Description+v.Instructions+v.Model) || strings.TrimSpace(v.Name) == "" || utf8.RuneCountInString(v.Name) > 120 || len(v.Description) > 4096 || len(v.Instructions) > 16384 || utf8.RuneCountInString(v.Model) > 200 || !core.ValidRole(v.Role) || v.Role == "human" {
		return core.Fail("validation", "Проверьте имя, роль и длину текстов профиля")
	}
	for _, r := range v.Name + v.Model {
		if unicode.IsControl(r) {
			return core.Fail("validation", "Имя и модель не должны содержать управляющие символы")
		}
	}
	// Multi-line texts follow the shared task-description rule.
	if !plainText(v.Description, true) || !plainText(v.Instructions, true) {
		return core.Fail("validation", "Описание и инструкции могут содержать переносы строк и табуляцию, но не другие управляющие символы")
	}
	switch v.Provider {
	case "unconfigured", "openai", "anthropic", "gennadiy", "local", "custom":
	default:
		return core.Fail("validation", "Выберите поставщика из списка")
	}
	return nil
}
func profileFor(b *board, id string) (*core.AgentProfile, error) {
	if !profileID.MatchString(id) {
		return nil, core.Fail("not_found", "Профиль не найден")
	}
	v := b.Profiles[id]
	if v == nil {
		return nil, core.Fail("not_found", "Профиль не найден")
	}
	return v, nil
}
func sessionProfile(b *board, in core.OpenSession) (*core.ProfileSnapshot, error) {
	if in.ProfileID == "" && in.ProfileVersion == 0 {
		return nil, nil
	}
	if in.ProfileID == "" || in.ProfileVersion < 1 {
		return nil, core.Fail("validation", "Нужны profile_id и положительная profile_version")
	}
	v, err := profileFor(b, in.ProfileID)
	if err != nil {
		return nil, err
	}
	if v.State != "active" || v.Version != in.ProfileVersion {
		return nil, core.Fail("conflict", "Профиль изменён или архивирован")
	}
	if in.Role != v.Role {
		return nil, core.Fail("validation", "Роль должна совпадать с профилем")
	}
	return &core.ProfileSnapshot{ProfileSettings: v.ProfileSettings, ID: v.ID, Version: v.Version}, nil
}
func (s *Store) Profiles(ctx context.Context, a core.Actor, in core.ProfilesInput) (core.ProfilesPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ProfilesPage, error) {
		page := core.ProfilesPage{Profiles: []core.AgentProfile{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		state := in.State
		if state == "" {
			state = "active"
		}
		if state != "all" && state != "active" && state != "archived" {
			return page, core.Fail("validation", "Неверное состояние профиля")
		}
		for _, v := range b.Profiles {
			if state == "all" || v.State == state {
				summary := *v
				summary.Instructions = ""
				page.Profiles = append(page.Profiles, summary)
			}
		}
		sort.Slice(page.Profiles, func(i, j int) bool { return page.Profiles[i].ID < page.Profiles[j].ID })
		return page, nil
	})
}
func (s *Store) Profile(ctx context.Context, a core.Actor, in core.ProfileInput) (core.ProfilePage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ProfilePage, error) {
		page := core.ProfilePage{Revisions: []core.ProfileRevision{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return page, err
		}
		v, err := profileFor(b, in.ProfileID)
		if err != nil {
			return page, err
		}
		if in.Version < 0 {
			return page, core.Fail("validation", "Неверная версия профиля")
		}
		page.Profile = *v
		if in.Version > 0 {
			var raw string
			if err = b.tx.QueryRowContext(ctx, `SELECT data FROM agent_profile_revisions WHERE profile_id=? AND version=?`, in.ProfileID, in.Version).Scan(&raw); err != nil {
				if err == sql.ErrNoRows {
					return page, core.Fail("not_found", "Версия профиля не найдена")
				}
				return page, err
			}
			if err = json.Unmarshal([]byte(raw), &page.Profile); err != nil {
				return page, err
			}
		}
		rows, err := b.tx.QueryContext(ctx, `SELECT data FROM agent_profile_revisions WHERE profile_id=? ORDER BY version DESC LIMIT 1024`, in.ProfileID)
		if err != nil {
			return page, err
		}
		defer rows.Close()
		for rows.Next() {
			var raw string
			var revision core.AgentProfile
			if err = rows.Scan(&raw); err != nil {
				return page, err
			}
			if err = json.Unmarshal([]byte(raw), &revision); err != nil {
				return page, err
			}
			page.Revisions = append(page.Revisions, core.ProfileRevision{Version: revision.Version, State: revision.State, UpdatedAt: revision.UpdatedAt, UpdatedBy: revision.UpdatedBy})
		}
		return page, rows.Err()
	})
}
func recordProfile(b *board, v core.AgentProfile, a core.Actor, now time.Time, kind string) core.AgentProfile {
	v.UpdatedAt = now
	v.UpdatedBy = a.SessionID
	b.Profiles[v.ID] = &v
	b.ProfileRevisions = append(b.ProfileRevisions, v)
	emit(b, now, "profile_"+kind, "", a.SessionID)
	b.Events[len(b.Events)-1].Reason = v.ID + "@" + strconv.FormatInt(v.Version, 10)
	return v
}
func (s *Store) ProfileCreate(ctx context.Context, a core.Actor, in core.ProfileCreateInput) (core.AgentProfile, error) {
	return mutate(s, ctx, a, in.RequestID, "profile_create", in, func(b *board, _ *core.Session, now time.Time) (core.AgentProfile, error) {
		if err := validateProfile(in.ProfileSettings); err != nil {
			return core.AgentProfile{}, err
		}
		if len(b.Profiles) >= 128 {
			return core.AgentProfile{}, core.Fail("conflict", "Достигнут предел 128 профилей")
		}
		var bytes [16]byte
		if _, err := rand.Read(bytes[:]); err != nil {
			return core.AgentProfile{}, err
		}
		v := core.AgentProfile{ProfileSettings: in.ProfileSettings, ID: "ap_" + hex.EncodeToString(bytes[:]), Version: 1, State: "active", CreatedAt: now, CreatedBy: a.SessionID}
		return recordProfile(b, v, a, now, "created"), nil
	})
}
func (s *Store) ProfileUpdate(ctx context.Context, a core.Actor, in core.ProfileUpdateInput) (core.AgentProfile, error) {
	return mutate(s, ctx, a, in.RequestID, "profile_update", in, func(b *board, _ *core.Session, now time.Time) (core.AgentProfile, error) {
		v, err := editableProfile(b, in.ProfileID, in.Version)
		if err != nil {
			return core.AgentProfile{}, err
		}
		if err = validateProfile(in.ProfileSettings); err != nil {
			return core.AgentProfile{}, err
		}
		next := *v
		next.ProfileSettings = in.ProfileSettings
		next.Version++
		return recordProfile(b, next, a, now, "updated"), nil
	})
}
func (s *Store) ProfileArchive(ctx context.Context, a core.Actor, in core.ProfileArchiveInput) (core.AgentProfile, error) {
	return mutate(s, ctx, a, in.RequestID, "profile_archive", in, func(b *board, _ *core.Session, now time.Time) (core.AgentProfile, error) {
		v, err := editableProfile(b, in.ProfileID, in.Version)
		if err != nil {
			return core.AgentProfile{}, err
		}
		if (in.State != "active" && in.State != "archived") || v.State == in.State {
			return core.AgentProfile{}, core.Fail("validation", "Укажите новое состояние active или archived")
		}
		next := *v
		next.State = in.State
		next.Version++
		return recordProfile(b, next, a, now, "state_changed"), nil
	})
}
func editableProfile(b *board, id string, version int64) (*core.AgentProfile, error) {
	v, err := profileFor(b, id)
	if err != nil {
		return nil, err
	}
	if version != v.Version || version < 1 {
		return nil, core.Fail("conflict", "Версия профиля изменилась; обновите форму")
	}
	if v.Version >= 1024 {
		return nil, core.Fail("conflict", "Достигнут предел 1024 ревизий профиля")
	}
	return v, nil
}
