package main

import (
	"agent-board/internal/store"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-board/internal/core"
)

func createToken(args []string, out io.Writer) error {
	f := flags("token create")
	db := f.String("db", ".local/board.db", "database path")
	owner := f.String("owner", "", "participant name")
	machine := f.String("machine", "", "machine name")
	roles := f.String("roles", "", "explicit comma-separated role scopes")
	days := f.Int("days", 30, "validity in days, 1..365")
	output := f.String("output", "", "new private output file; never overwritten")
	if err := parse(f, args); err != nil {
		return err
	}
	if *owner == "" || *machine == "" || *output == "" || *days < 1 || *days > 365 {
		return errors.New("owner, machine, output and days 1..365 are required")
	}
	parent, err := os.Stat(filepath.Dir(*output))
	if err != nil || !parent.IsDir() || parent.Mode().Perm() != 0700 {
		return errors.New("token output parent directory must exist with mode 0700")
	}
	file, err := os.OpenFile(*output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return errors.New("cannot create token output; choose a new private file")
	}
	completed := false
	defer func() {
		_ = file.Close()
		if !completed {
			_ = os.Remove(*output)
		}
	}()
	s, err := store.Open(*db, core.Config{Project: "agent-board", LeaseTTL: 15 * time.Minute, SessionTTL: 15 * time.Minute, BootstrapUsers: []string{*owner}}, nil)
	if err != nil {
		return err
	}
	defer s.Close()
	var roleNames []string
	if *roles != "" {
		roleNames = strings.Split(*roles, ",")
	}
	token, err := s.CreateToken(context.Background(), core.TokenInput{Roles: roleNames, Owner: *owner, Machine: *machine, TTL: time.Duration(*days) * 24 * time.Hour})
	if err != nil {
		return err
	}
	if _, err = io.WriteString(file, token.Secret+"\n"); err == nil {
		err = file.Sync()
	}
	if err == nil {
		err = file.Close()
	}
	if err != nil {
		if revokeErr := s.RevokeToken(context.Background(), token.ID); revokeErr != nil {
			return fmt.Errorf("token %s output failed and revocation failed; revoke this ID before retrying", token.ID)
		}
		return errors.New("token output failed; token was revoked; choose a new output and retry")
	}
	completed = true
	_, err = fmt.Fprintf(out, "Token %s saved to %s (expires %s). Keep this file private.\n", token.ID, *output, token.ExpiresAt.UTC().Format(time.RFC3339))
	return err
}

func revokeToken(args []string, out io.Writer) error {
	f := flags("token revoke")
	db := f.String("db", ".local/board.db", "database path")
	id := f.String("id", "", "token ID to revoke")
	if err := parse(f, args); err != nil {
		return err
	}
	if *id == "" {
		return errors.New("token ID is required")
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.RevokeToken(context.Background(), *id); err != nil {
		return errors.New("token revocation failed")
	}
	_, err = fmt.Fprintln(out, "Token revoked.")
	return err
}
