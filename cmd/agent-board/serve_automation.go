package main

import (
	"agent-board/internal/projects"
	"context"
	"errors"
)

// The loop must exit before registry.Close. An unexpected storage error stops
// admission rather than leaving a healthy-looking server with a dead scheduler.
func startAutomation(ctx context.Context, stop context.CancelFunc, r *projects.Registry) <-chan error {
	done := make(chan error, 1)
	if r == nil {
		close(done)
		return done
	}
	go func() {
		defer close(done)
		err := r.RunAutomation(ctx)
		if err != nil && !errors.Is(err, context.Canceled) {
			done <- errors.New("automation scheduler stopped unexpectedly")
			stop()
		}
	}()
	return done
}
