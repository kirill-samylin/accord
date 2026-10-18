package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"slices"
	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
	"agent-board/internal/store"
)

func configCommand(args []string, out io.Writer) error {
	if len(args) == 0 {
		return errors.New("usage: config init|validate|get|history|apply")
	}
	f := flags("config " + args[0])
	switch args[0] {
	case "init":
		profile := f.String("profile", "davinci", "configuration profile")
		if err := parse(f, args[1:]); err != nil {
			return err
		}
		if *profile != "davinci" {
			return errors.New("supported profile: davinci")
		}
		_, err := out.Write(config.DavinciTemplate())
		return err
	case "validate":
		// The file is positional; optional profile can precede or follow it.
		profile := f.String("profile", "", "profile hint")
		fileArgs := args[1:]
		path := ""
		if len(fileArgs) > 0 && fileArgs[0] != "--profile" {
			path = fileArgs[0]
			fileArgs = fileArgs[1:]
		}
		if err := f.Parse(fileArgs); err != nil {
			return err
		}
		if path == "" && f.NArg() == 1 {
			path = f.Arg(0)
		} else if f.NArg() != 0 {
			return errors.New("one configuration file required")
		}
		if *profile != "" && *profile != "davinci" {
			return errors.New("unknown profile")
		}
		raw, err := readConfigFile(path)
		if err != nil {
			return err
		}
		_, result := config.Parse(raw, config.Options{})
		if err := json.NewEncoder(out).Encode(result); err != nil {
			return err
		}
		if !result.Valid {
			return errors.New("configuration validation failed")
		}
		return nil
	case "get", "history", "apply":
		db := f.String("db", ".local/board.db", "private database")
		file := f.String("file", "", "configuration file")
		expected := f.Int64("expected-version", -1, "current configuration CAS version")
		actor := f.String("actor", "", "local administrator label")
		after := f.Int64("after-version", 0, "exclusive history version")
		limit := f.Int("limit", 100, "history page 1..200")
		owner := f.String("bootstrap-owner", "", "explicit initial local user")
		project := f.String("project", "", "actual pre-migration project")
		taskTTL := f.Duration("task-ttl", 0, "actual pre-migration task TTL")
		sessionTTL := f.Duration("session-ttl", 0, "actual pre-migration session/role TTL")
		if err := parse(f, args[1:]); err != nil {
			return err
		}
		if args[0] != "apply" {
			s, err := existingDB(*db)
			if err != nil {
				return err
			}
			defer s.Close()
			if args[0] == "get" {
				v, err := s.Config(context.Background())
				if err != nil {
					return err
				}
				return json.NewEncoder(out).Encode(v)
			}
			v, err := s.ConfigHistory(context.Background(), *after, *limit)
			if err != nil {
				return err
			}
			return json.NewEncoder(out).Encode(v)
		}
		raw, err := readConfigFile(*file)
		if err != nil {
			return err
		}
		candidate, validation := config.Parse(raw, config.Options{})
		if !validation.Valid {
			if err := json.NewEncoder(out).Encode(validation); err != nil {
				return err
			}
			return errors.New("configuration validation failed")
		}
		if *expected < 0 || *actor == "" {
			return errors.New("expected-version and actor required")
		}
		var s *store.Store
		_, statErr := os.Stat(*db)
		if os.IsNotExist(statErr) {
			if *owner == "" || *project == "" || *taskTTL < time.Second || *sessionTTL < time.Second || *expected != 0 || !slices.Contains(candidate.Decode().Auth.Local.Users, *owner) {
				return errors.New("initial apply requires expected-version 0, bootstrap-owner, project, task-ttl and session-ttl")
			}
			s, err = store.Open(*db, core.Config{SkipBootstrap: true, BootstrapUsers: []string{*owner}, Project: *project, LeaseTTL: *taskTTL, SessionTTL: *sessionTTL}, nil)
		} else {
			schema, e := store.DatabaseSchema(context.Background(), *db)
			if e != nil {
				return e
			}
			if schema == 0 {
				if *owner == "" || *project == "" || *taskTTL < time.Second || *sessionTTL < time.Second || *expected != 0 || !slices.Contains(candidate.Decode().Auth.Local.Users, *owner) {
					return errors.New("initial apply requires expected-version 0, allowlisted bootstrap-owner, project and explicit TTLs")
				}
				s, err = store.Open(*db, core.Config{SkipBootstrap: true, BootstrapUsers: []string{*owner}, Project: *project, LeaseTTL: *taskTTL, SessionTTL: *sessionTTL}, nil)
			} else if schema < 3 {
				if *owner == "" || *project == "" || *taskTTL < time.Second || *sessionTTL < time.Second {
					return errors.New("migration requires explicit bootstrap-owner, project, task-ttl and session-ttl")
				}
				s, err = store.Open(*db, core.Config{BootstrapUsers: []string{*owner}, Project: *project, LeaseTTL: *taskTTL, SessionTTL: *sessionTTL}, nil)
			} else if schema < store.SchemaVersion {
				s, err = store.Open(*db, core.Config{}, nil)
			} else {
				if *expected == 0 {
					if *owner == "" || *project == "" || *taskTTL < time.Second || *sessionTTL < time.Second {
						return errors.New("initial apply requires explicit bootstrap owner, project and TTLs")
					}
					s, err = store.Open(*db, core.Config{SkipBootstrap: true, BootstrapUsers: []string{*owner}, Project: *project, LeaseTTL: *taskTTL, SessionTTL: *sessionTTL}, nil)
				} else {
					s, err = existingDB(*db)
				}
			}
		}
		if err != nil {
			return err
		}
		defer s.Close()
		result, err := s.ApplyConfig(context.Background(), raw, *expected, *actor)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(result)
	}
	return errors.New("unknown config command")
}
func readConfigFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, config.MaxInputBytes+1))
}
