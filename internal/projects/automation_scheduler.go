package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"errors"
	"time"
)

// RunAutomation owns no credentials or native processes. Each transition
// reacquires registry -> legacy -> board authority; SQL/CAS fences all servers.
// The caller cancels and joins this loop before closing the registry. Only a
// registry-wide failure (listing projects) stops it; a failing project is
// logged without secrets and backed off while the others continue.
func (r *Registry) RunAutomation(ctx context.Context) error {
	if r.options.Offline {
		return errors.New("automation scheduler is unavailable offline")
	}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	start := 0
	cursors := map[string]int{}
	for {
		cycle, cancel := context.WithTimeout(ctx, time.Second)
		next, err := r.automationCycle(cycle, start, cursors)
		cancel()
		start = next
		if err != nil && !errors.Is(err, context.DeadlineExceeded) && !errors.Is(err, context.Canceled) {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (r *Registry) automationProjectIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM projects WHERE state IN ('active','archived') ORDER BY id LIMIT ?`, MaxProjects)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Registry) automationCycle(ctx context.Context, start int, cursors map[string]int) (int, error) {
	ids, err := r.automationProjectIDs(ctx)
	if err != nil || len(ids) == 0 {
		return start, err
	}
	start %= len(ids)
	budget := 32
	next := start
	for offset := 0; offset < len(ids) && budget > 0; offset++ {
		projectIndex := (start + offset) % len(ids)
		projectID := ids[projectIndex]
		next = (projectIndex + 1) % len(ids)
		if err = ctx.Err(); err != nil {
			return next, err
		}
		if r.backoff.skip(projectID, time.Now()) {
			continue
		}
		v, e := r.runtime(projectID)
		if e != nil {
			r.backoff.fail(projectID, e, time.Now())
			continue
		}
		candidates, e := transaction(r, ctx, func(tx *sql.Tx) ([]core.WorkflowAdvanceInput, error) {
			project, err := r.project(ctx, tx, projectID)
			if err != nil {
				return nil, err
			}
			return automationGuard(v.service, ctx, tx, project, core.Principal{}, nil, func(inner context.Context) ([]core.WorkflowAdvanceInput, error) {
				return v.store.AutomationCandidates(inner, 32)
			})
		})
		if e != nil {
			if cycleExpired(ctx, e) {
				return next, e
			}
			r.backoff.fail(projectID, e, time.Now())
			continue
		}
		failed := false
		batch := min(4, budget, len(candidates))
		for position := 0; position < batch; position++ {
			candidateIndex := cursors[projectID] % len(candidates)
			candidate := candidates[candidateIndex]
			cursors[projectID] = (candidateIndex + 1) % len(candidates)
			budget--
			_, e = transaction(r, ctx, func(tx *sql.Tx) (core.WorkflowAdvanceResult, error) {
				p, e := r.project(ctx, tx, projectID)
				if e != nil {
					return core.WorkflowAdvanceResult{}, e
				}
				return automationGuard(v.service, ctx, tx, p, core.Principal{}, nil, func(inner context.Context) (core.WorkflowAdvanceResult, error) {
					return v.store.AutomationTick(inner, candidate)
				})
			})
			if e != nil {
				var domain *core.Error
				// An ordinary CAS/access race is not an infrastructure failure.
				// It never grants dispatch; the next scan observes current state.
				if !errors.As(e, &domain) {
					if cycleExpired(ctx, e) {
						return next, e
					}
					// One broken project (storage, migration, damaged board) is
					// isolated and backed off; other projects keep running.
					r.backoff.fail(projectID, e, time.Now())
					failed = true
					break
				}
			}
		}
		if !failed {
			r.backoff.ok(projectID)
		}
	}
	return next, nil
}
