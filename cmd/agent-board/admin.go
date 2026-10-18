package main

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"context"
	"encoding/json"
	"errors"
	"io"
)

func existingDB(path string) (*store.Store, error) {
	s, err := store.OpenExisting(path, core.Config{Project: "agent-board"}, nil)
	if err != nil {
		return nil, errors.New("cannot open existing private board database")
	}
	return s, nil
}

func backup(args []string, out io.Writer) error {
	f := flags("backup")
	db := f.String("db", ".local/board.db", "existing source database")
	output := f.String("output", "", "new snapshot file in a private mode 0700 directory")
	if err := parse(f, args); err != nil {
		return err
	}
	if *output == "" {
		return errors.New("backup output is required")
	}
	result, err := store.Backup(context.Background(), *db, *output)
	if err != nil {
		return errors.New("backup failed; source must exist and output must be new and private")
	}
	return json.NewEncoder(out).Encode(result)
}

func listTokens(args []string, out io.Writer) error {
	f := flags("token list")
	db := f.String("db", ".local/board.db", "existing database")
	if err := parse(f, args); err != nil {
		return err
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	tokens, err := s.ListTokens(context.Background())
	if err != nil {
		return errors.New("cannot list tokens")
	}
	return json.NewEncoder(out).Encode(tokens)
}

func revokeAllTokens(args []string, out io.Writer) error {
	f := flags("token revoke-all")
	db := f.String("db", ".local/board.db", "existing database")
	reason := f.String("reason", "", "required administrative reason")
	if err := parse(f, args); err != nil {
		return err
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	count, err := s.RevokeAllTokens(context.Background(), *reason)
	if err != nil {
		return errors.New("token revocation failed; supply a non-empty bounded reason")
	}
	return json.NewEncoder(out).Encode(struct {
		Revoked int `json:"revoked_tokens"`
	}{count})
}

func recoverState(args []string, out io.Writer) error {
	f := flags("maintenance reset")
	db := f.String("db", ".local/board.db", "restored database; stop every server before this offline operation")
	reason := f.String("reason", "restore", "must be restore")
	if err := parse(f, args); err != nil {
		return err
	}
	if *reason != "restore" {
		return errors.New("maintenance reset requires reason restore")
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	result, err := s.RecoverState(context.Background(), *reason)
	if err != nil {
		return errors.New("offline recovery failed")
	}
	return json.NewEncoder(out).Encode(result)
}

func listSessions(args []string, out io.Writer) error {
	f := flags("sessions list")
	db := f.String("db", ".local/board.db", "existing database")
	if err := parse(f, args); err != nil {
		return err
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	sessions, err := s.ListSessions(context.Background())
	if err != nil {
		return errors.New("cannot list sessions")
	}
	return json.NewEncoder(out).Encode(sessions)
}
