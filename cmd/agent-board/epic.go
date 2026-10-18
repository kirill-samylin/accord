package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
)

func epicCommand(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "handoff" {
		return errors.New("usage: epic handoff --db DB --key E-N --owner NAME --expected-version N --reason TEXT --actor LABEL")
	}
	f := flags("epic handoff")
	db := f.String("db", ".local/board.db", "database")
	key := f.String("key", "", "epic key")
	owner := f.String("owner", "", "new existing owner")
	version := f.Int64("expected-version", 0, "epic CAS version")
	reason := f.String("reason", "", "handoff reason")
	actor := f.String("actor", "", "local administrator")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	v, err := s.HandoffEpic(context.Background(), *key, *owner, *version, *reason, *actor)
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(v)
}
