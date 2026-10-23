package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"time"

	"agent-board/internal/worker"
)

func workerAutomationPreview(args []string, out io.Writer) error {
	f := flags("worker automation-preview")
	path := f.String("config", "", "private local draft; preview never applies consent")
	if err := parse(f, args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("automation-preview requires --config PRIVATE")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	proposal, err := worker.PreviewAutomation(ctx, *path)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(proposal)
}
