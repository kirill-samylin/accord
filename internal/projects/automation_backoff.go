package projects

import (
	"agent-board/internal/core"
	"context"
	"errors"
	"log"
	"strings"
	"sync"
	"time"
	"unicode"
)

const automationBackoffMin = 2 * time.Second
const automationBackoffMax = 5 * time.Minute

// automationBackoff remembers per-project scheduler failures. A failing project
// is retried with exponential delay; success resets it. It grants nothing: a
// skipped project simply is not advanced until its next attempt.
type automationBackoff struct {
	mu       sync.Mutex
	projects map[string]automationBackoffEntry
}
type automationBackoffEntry struct {
	until    time.Time
	failures int
}

func (b *automationBackoff) skip(id string, now time.Time) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	e, ok := b.projects[id]
	return ok && now.Before(e.until)
}
func (b *automationBackoff) ok(id string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.projects, id)
}
func (b *automationBackoff) fail(id string, err error, now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.projects == nil {
		b.projects = map[string]automationBackoffEntry{}
	}
	e := b.projects[id]
	delay := automationBackoffMin
	for i := 0; i < e.failures && delay < automationBackoffMax; i++ {
		delay *= 2
	}
	delay = min(delay, automationBackoffMax)
	e.failures++
	e.until = now.Add(delay)
	b.projects[id] = e
	log.Printf("automation scheduler: project %s skipped for %s after failure %d: %s", id, delay, e.failures, schedulerErrorSummary(err))
}

// schedulerErrorSummary keeps logs useful without secrets: domain errors log
// their code, others a bounded single-line message (no request payloads).
func schedulerErrorSummary(err error) string {
	var domain *core.Error
	if errors.As(err, &domain) {
		return domain.Code
	}
	text := strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, err.Error())
	if len(text) > 200 {
		text = text[:200]
	}
	return text
}

// cycleExpired distinguishes the scan's own time budget from a project fault.
func cycleExpired(ctx context.Context, err error) bool {
	return ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)
}
