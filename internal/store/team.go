package store

import (
	"context"
	"sort"
	"time"

	"agent-board/internal/core"
)

const teamWindow = 24 * time.Hour

// ControlTeam groups recorded participation by owner label in one
// transaction: who is online, which agent sessions hold which tasks, what each
// person created or finished recently, and suspected duplicate pairs.
// Agent sessions are board participation, not OS process liveness.
func (s *Store) ControlTeam(ctx context.Context, a core.Actor) (core.TeamView, error) {
	return transact(s, ctx, func(b *board, now time.Time) (core.TeamView, error) {
		mode, threshold := duplicateMode(b)
		view := core.TeamView{ObservedAt: now, Project: b.Config.Project, RecentWindowHours: int(teamWindow / time.Hour), Members: []core.TeamMember{}, Unclaimed: []core.Task{}, Duplicates: []core.DuplicatePair{}, DuplicateMode: mode, DuplicateThreshold: threshold}
		if err := readActor(ctx, b, a, now); err != nil {
			return view, err
		}
		credentials, err := controlCredentials(ctx, b, now)
		if err != nil {
			return view, err
		}
		lastEvents, err := controlLastEvents(ctx, b)
		if err != nil {
			return view, err
		}
		members := map[string]*core.TeamMember{}
		member := func(owner string) *core.TeamMember {
			if owner == "" {
				owner = "—"
			}
			m := members[owner]
			if m == nil {
				m = &core.TeamMember{Owner: owner, Machines: []string{}, Agents: []core.ControlSession{}}
				members[owner] = m
			}
			return m
		}
		held := map[string]*core.Task{}
		for _, t := range b.Tasks {
			if controlLease(b, t, credentials, now) {
				v := viewTask(b, t)
				held[t.Lease.SessionID] = &v
			}
		}
		machines := map[string]map[string]bool{}
		for _, sess := range b.Sessions {
			owner := sess.Owner
			if owner == "" {
				owner = credentials[sess.PrincipalID].owner
			}
			if last := lastEvents[sess.SessionID]; last != nil && now.Sub(*last) < teamWindow {
				m := member(owner)
				if m.LastEventAt == nil || last.After(*m.LastEventAt) {
					m.LastEventAt = last
				}
			}
			if controlStatus(sess, credentials, now) != "active" {
				continue
			}
			m := member(owner)
			if machines[m.Owner] == nil {
				machines[m.Owner] = map[string]bool{}
			}
			machines[m.Owner][sess.Machine] = true
			if sess.Role == "human" {
				m.HumanOnline = true
				continue
			}
			profile := sess.Profile
			if profile != nil {
				summary := *profile
				summary.Instructions = ""
				profile = &summary
			}
			m.Agents = append(m.Agents, core.ControlSession{Profile: profile, SessionID: sess.SessionID, Role: sess.Role, RunID: sess.RunID, Machine: sess.Machine, Owner: owner, RoleGeneration: sess.RoleGeneration, ExpiresAt: sess.ExpiresAt, Status: "active", Task: held[sess.SessionID], LastEventAt: lastEvents[sess.SessionID]})
			if held[sess.SessionID] != nil {
				m.InProgress++
			}
		}
		for _, t := range b.Tasks {
			if t.Type != "task" {
				continue
			}
			v := core.Task{Requester: t.Requester, CreatedBy: t.CreatedBy}
			projectPeople(b, &v)
			if v.CreatedBy != "" {
				if t.State == "created" || t.State == "ready" || t.State == "in_progress" {
					member(v.CreatedBy).OpenRequested++
				}
				if t.CreatedAt != nil && now.Sub(*t.CreatedAt) < teamWindow {
					member(v.CreatedBy).CreatedRecent++
				}
			}
			if t.State == "done" {
				if at := currentAttempt(t); at != nil && at.EndedAt != nil && now.Sub(*at.EndedAt) < teamWindow {
					if author := b.Sessions[t.AuthorSession]; author != nil {
						member(author.Owner).DoneRecent++
					}
				}
			}
			if t.State == "ready" && t.Lease == nil && !blocked(b, t) && assignmentAvailable(b, t) && t.Role != "human" {
				epic := b.Tasks[t.EpicKey]
				if epic != nil && epic.State == "open" {
					view.UnclaimedTotal++
					view.Unclaimed = append(view.Unclaimed, viewTask(b, t))
				}
			}
		}
		sort.Slice(view.Unclaimed, func(i, j int) bool {
			x, y := view.Unclaimed[i], view.Unclaimed[j]
			if x.Priority != y.Priority {
				return x.Priority < y.Priority
			}
			return x.ReadyAt.Before(y.ReadyAt)
		})
		if len(view.Unclaimed) > 8 {
			view.Unclaimed = view.Unclaimed[:8]
		}
		for owner, set := range machines {
			for machine := range set {
				members[owner].Machines = append(members[owner].Machines, machine)
			}
			sort.Strings(members[owner].Machines)
		}
		for _, m := range members {
			sort.Slice(m.Agents, func(i, j int) bool {
				if (m.Agents[i].Task != nil) != (m.Agents[j].Task != nil) {
					return m.Agents[i].Task != nil
				}
				return m.Agents[i].SessionID < m.Agents[j].SessionID
			})
			view.Members = append(view.Members, *m)
		}
		sort.Slice(view.Members, func(i, j int) bool {
			x, y := view.Members[i], view.Members[j]
			xo, yo := len(x.Agents) > 0 || x.HumanOnline, len(y.Agents) > 0 || y.HumanOnline
			if xo != yo {
				return xo
			}
			if x.InProgress != y.InProgress {
				return x.InProgress > y.InProgress
			}
			return x.Owner < y.Owner
		})
		if mode != core.DuplicateOff {
			pairs := duplicatePairs(b, now, threshold)
			view.DuplicatesTotal = len(pairs)
			if len(pairs) > 20 {
				pairs = pairs[:20]
			}
			view.Duplicates = pairs
		}
		return view, nil
	})
}
