package store

import (
	"agent-board/internal/core"
	"context"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func managedValues(values []string, max int, providers bool) error {
	if len(values) == 0 || len(values) > max {
		return core.Fail("validation", "Нужен непустой ограниченный список")
	}
	seen := map[string]bool{}
	for _, v := range values {
		if !bounded(v, 128) || seen[v] || strings.ContainsAny(v, "/\\\x00\r\n\t") {
			return core.Fail("validation", "Неверное или повторное локальное разрешение")
		}
		for _, r := range v {
			if unicode.IsControl(r) {
				return core.Fail("validation", "Управляющие символы запрещены")
			}
		}
		if providers && v != "codex" && v != "claude" {
			return core.Fail("validation", "Поддерживаются codex/claude")
		}
		seen[v] = true
	}
	return nil
}
func managedEnrollValid(in core.ManagedHostEnrollInput) error {
	if !bounded(in.Name, 120) || !utf8.ValidString(in.Name) {
		return core.Fail("validation", "Нужно имя компьютера до120байт")
	}
	if secret, err := hex.DecodeString(in.HostSecret); err != nil || len(secret) != 32 {
		return core.Fail("validation", "Нужен случайный256bit локальный host proof")
	}
	if len(in.Controllers) > 128 {
		return core.Fail("validation", "Слишком много контроллеров")
	}
	if len(in.Controllers) > 0 {
		if err := managedValues(in.Controllers, 128, false); err != nil {
			return err
		}
	}
	if err := managedValues(in.Workspaces, 16, false); err != nil {
		return err
	}
	return managedValues(in.Providers, 2, true)
}
func (s *Store) ManagedHostEnroll(ctx context.Context, a core.Actor, in core.ManagedHostEnrollInput) (core.ManagedHost, error) {
	if err := managedEnrollValid(in); err != nil {
		return core.ManagedHost{}, err
	}
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	return managedMutate(s, ctx, a, in.RequestID, "host_enroll", in, func(*board, time.Time) error { return nil }, func(b *board, now time.Time) (core.ManagedHost, error) {
		if len(b.ManagedHosts) >= 32 {
			return core.ManagedHost{}, core.Fail("conflict", "Достигнут предел32компьютера")
		}
		id, err := managedID("mh_")
		if err != nil {
			return core.ManagedHost{}, err
		}
		h := &managedHostRecord{SecretHash: managedHash(secret), Host: core.ManagedHost{ID: id, OwnerPrincipalID: a.Principal.ID, OwnerAccessEpoch: managedEpoch(ctx), Name: in.Name, State: "active", Version: 1, Controllers: in.Controllers, Workspaces: in.Workspaces, Providers: in.Providers}}
		b.ManagedHosts[id] = h
		emit(b, now, "managed_host_enrolled", "", a.SessionID)
		return managedHostView(h, a), nil
	})
}
func (s *Store) ManagedHostUpdate(ctx context.Context, a core.Actor, in core.ManagedHostUpdateInput) (core.ManagedHost, error) {
	if err := managedEnrollValid(in.ManagedHostEnrollInput); err != nil {
		return core.ManagedHost{}, err
	}
	secret := in.HostSecret
	in.HostSecret = managedHash(secret)
	check := func(b *board, _ time.Time) error {
		h, err := managedAllowed(b, a, in.HostID, "", true)
		if err == nil && (!managedProof(h, secret) || h.Host.OwnerAccessEpoch != managedEpoch(ctx)) {
			return core.Fail("unauthorized", "Локальное разрешение требует повторного подключения")
		}
		return err
	}
	return managedMutate(s, ctx, a, in.RequestID, "host_update", in, check, func(b *board, now time.Time) (core.ManagedHost, error) {
		h := b.ManagedHosts[in.HostID]
		if h.Host.Version != in.Version {
			return core.ManagedHost{}, core.Fail("conflict", "Версия компьютера изменилась")
		}
		if in.State != "active" && in.State != "disabled" {
			return core.ManagedHost{}, core.Fail("validation", "Неверное состояние компьютера")
		}
		for _, r := range b.ManagedRuns {
			if r.HostID == h.Host.ID && managedActive(r.State) && (!reflect.DeepEqual(in.Workspaces, h.Host.Workspaces) || !reflect.DeepEqual(in.Providers, h.Host.Providers)) {
				return core.ManagedHost{}, core.Fail("conflict", "Сначала остановите все процессы компьютера")
			}
		}
		h.Host.Name = in.Name
		h.Host.State = in.State
		h.Host.Controllers = in.Controllers
		h.Host.Workspaces = in.Workspaces
		h.Host.Providers = in.Providers
		h.Host.Version++
		emit(b, now, "managed_host_updated", "", a.SessionID)
		return managedHostView(h, a), nil
	})
}
func (s *Store) ManagedHostDisable(ctx context.Context, a core.Actor, in core.ManagedHostDisableInput) (core.ManagedHost, error) {
	return managedMutate(s, ctx, a, in.RequestID, "host_disable", in, func(b *board, _ time.Time) error { _, e := managedAllowed(b, a, in.HostID, "", false); return e }, func(b *board, now time.Time) (core.ManagedHost, error) {
		h := b.ManagedHosts[in.HostID]
		if h.Host.Version != in.Version {
			return core.ManagedHost{}, core.Fail("conflict", "Версия компьютера изменилась")
		}
		h.Host.State = "disabled"
		h.Host.Version++
		emit(b, now, "managed_host_disabled", "", a.SessionID)
		return managedHostView(h, a), nil
	})
}
func (s *Store) ManagedList(ctx context.Context, a core.Actor, in core.ManagedListInput) (core.ManagedPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ManagedPage, error) {
		out := core.ManagedPage{ObservedAt: now, ActorPrincipalID: a.Principal.ID, Hosts: []core.ManagedHost{}, Runs: []core.ManagedRun{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return out, err
		}
		if in.HostID != "" {
			if _, err := managedHost(b, in.HostID); err != nil {
				return out, err
			}
		}
		for _, h := range b.ManagedHosts {
			if in.HostID == "" || in.HostID == h.Host.ID {
				out.Hosts = append(out.Hosts, managedHostView(h, a))
			}
		}
		for _, r := range b.ManagedRuns {
			if in.HostID == "" || r.HostID == in.HostID {
				v := *r
				v.Prompt = ""
				v.Output = []string{}
				if r.Profile != nil {
					p := *r.Profile
					p.Instructions = ""
					p.Description = ""
					v.Profile = &p
				}
				out.Runs = append(out.Runs, v)
			}
		}
		sort.Slice(out.Hosts, func(i, j int) bool { return out.Hosts[i].ID < out.Hosts[j].ID })
		sort.Slice(out.Runs, func(i, j int) bool { return out.Runs[i].CreatedAt.After(out.Runs[j].CreatedAt) })
		return out, nil
	})
}
func (s *Store) ManagedRun(ctx context.Context, a core.Actor, in core.ManagedRunInput) (core.ManagedRunPage, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ManagedRunPage, error) {
		out := core.ManagedRunPage{Commands: []core.ManagedCommand{}}
		if err := readActor(ctx, b, a, now); err != nil {
			return out, err
		}
		r, err := managedRun(b, in.RunID)
		if err != nil {
			return out, err
		}
		out.Run = *r
		for _, c := range b.ManagedCommands {
			if c.Command.RunID == r.ID {
				out.Commands = append(out.Commands, c.Command)
			}
		}
		sort.Slice(out.Commands, func(i, j int) bool { return out.Commands[i].Sequence < out.Commands[j].Sequence })
		start, bytes := len(out.Commands), 0
		for start > 0 && len(out.Commands)-start < 100 {
			raw, err := json.Marshal(out.Commands[start-1])
			if err != nil {
				return out, err
			}
			if bytes+len(raw) > 256*1024 {
				break
			}
			bytes += len(raw)
			start--
		}
		if start > 0 {
			out.Commands = out.Commands[start:]
			out.CommandsTruncated = true
		}
		return out, nil
	})
}
func (s *Store) ManagedPermission(ctx context.Context, a core.Actor, in core.ManagedPermissionInput) (core.ManagedHost, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ManagedHost, error) {
		if err := managedActor(ctx, b, a, now); err != nil {
			return core.ManagedHost{}, err
		}
		h, err := managedAllowed(b, a, in.HostID, in.RunID, false)
		if err != nil {
			return core.ManagedHost{}, err
		}
		if !managedPrincipalAllowed(ctx, b, h.Host.OwnerPrincipalID, h.Host.OwnerAccessEpoch, now) {
			return core.ManagedHost{}, core.Fail("unauthorized", "Владелец компьютера потерял доступ")
		}
		if h.Host.State != "active" && in.Kind != "stop" && in.Kind != "disable" {
			return core.ManagedHost{}, core.Fail("forbidden", "Компьютер отключён")
		}
		return managedHostView(h, a), nil
	})
}

// ManagedAuthority is used under the registry guard to reject stale host-owner
// epochs. It does not renew a native session or reveal the stored proof hash.
func (s *Store) ManagedAuthority(ctx context.Context, a core.Actor, host, run string) (core.ManagedHost, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.ManagedHost, error) {
		if err := managedActor(ctx, b, a, now); err != nil {
			return core.ManagedHost{}, err
		}
		if run != "" {
			r, err := managedRun(b, run)
			if err != nil {
				return core.ManagedHost{}, err
			}
			host = r.HostID
		}
		h, err := managedHost(b, host)
		if err != nil {
			return core.ManagedHost{}, err
		}
		if !managedPrincipalAllowed(ctx, b, h.Host.OwnerPrincipalID, h.Host.OwnerAccessEpoch, now) {
			return core.ManagedHost{}, core.Fail("unauthorized", "Владелец компьютера потерял доступ")
		}
		return managedHostView(h, a), nil
	})
}
