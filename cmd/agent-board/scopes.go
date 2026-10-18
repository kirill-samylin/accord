package main

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
)

func tokenScopes(args []string, out io.Writer) error {
	if len(args) == 0 || args[0] != "set" {
		return errors.New("usage: token scopes set --db DB --id ID --roles LIST --expected-version N --actor LABEL")
	}
	f := flags("token scopes set")
	db := f.String("db", ".local/board.db", "database")
	id := f.String("id", "", "token ID")
	roles := f.String("roles", "", "explicit comma-separated role names")
	version := f.Int64("expected-version", 0, "scopes CAS version")
	actor := f.String("actor", "", "local administrator")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	if *id == "" || *roles == "" {
		return errors.New("id and roles required")
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	v, err := s.SetTokenScopes(context.Background(), core.ScopesInput{ID: *id, Roles: strings.Split(*roles, ","), ExpectedVersion: *version, Actor: *actor})
	if err != nil {
		return err
	}
	return json.NewEncoder(out).Encode(v)
}
