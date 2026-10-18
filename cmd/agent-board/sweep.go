package main

import (
	"agent-board/internal/store"
	"context"
	"time"
)

func sweep(ctx context.Context, s *store.Store) {
	for {
		interval, err := s.Sweep(ctx)
		if err != nil {
			interval = time.Second
		}
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
