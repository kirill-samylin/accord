package main

import (
	"agent-board/internal/core"
	"agent-board/internal/projects"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// All project administration is explicit, local and offline. Bearers are only
// written to new private files; command output contains a public identity ID.
func projectCommand(args []string, out io.Writer) error {
	if len(args) < 1 {
		return errors.New("usage: projects token-create | bootstrap-owner | bootstrap-admin | people-list")
	}
	operation := args[0]
	switch operation {
	case "token-create", "bootstrap-owner", "bootstrap-admin", "people-list", "maintenance-reset", "recover-owner", "token-revoke", "reject-provisioning", "migrate-storage":
	default:
		return errors.New("unknown projects operation")
	}
	f := flags("projects " + operation)
	root := f.String("projects-data", os.Getenv("ACCORD_PROJECTS_DATA"), "private registry root")
	db := f.String("db", ".local/board.db", "existing default board")
	webData := f.String("web-data", os.Getenv("ACCORD_WEB_DATA"), "bound default web root")
	owner := f.String("owner", "", "owner label")
	machine := f.String("machine", "", "machine label")
	days := f.Int("days", 30, "validity 1..365 days")
	output := f.String("output", "", "new private token file")
	reason := f.String("reason", "", "explicit restore or owner-recovery")
	project := f.String("project", "", "project for offline owner recovery")
	identity := f.String("identity", "", "explicit existing legacy identity ID")
	person := f.String("person", "", "existing person u_... for the new credential")
	if err := parse(f, args[1:]); err != nil {
		return err
	}
	if *root == "" || *webData == "" {
		return errors.New("projects-data and web-data required")
	}
	if operation == "migrate-storage" {
		count, err := projects.MigrateStorage(context.Background(), *root, *db, *webData)
		if err != nil {
			return errors.New("offline storage migration failed; keep startup guard and inspect")
		}
		return json.NewEncoder(out).Encode(map[string]any{"storage_migrated": true, "boards": count})
	}
	if operation == "reject-provisioning" {
		if *reason != "provisioning-recovery" {
			return errors.New("reject-provisioning requires --reason provisioning-recovery")
		}
		if err := projects.RejectProvisioning(context.Background(), *root, *project); err != nil {
			return errors.New("offline provisioning rejection failed; inspect private recovery state")
		}
		return json.NewEncoder(out).Encode(map[string]bool{"provisioning_rejected": true})
	}
	if operation == "maintenance-reset" {
		if *reason != "restore" {
			return errors.New("maintenance-reset requires --reason restore")
		}
		if err := projects.ResetOffline(context.Background(), *root, *db, *webData); err != nil {
			return errors.New("offline project restoration reset failed; do not serve staging")
		}
		return json.NewEncoder(out).Encode(map[string]bool{"restored_state_reset": true})
	}
	s, err := existingDB(*db)
	if err != nil {
		return err
	}
	defer s.Close()
	r, err := projects.Open(projects.Options{DataDir: *root, DefaultDB: *db, DefaultWeb: *webData, Default: s, Offline: true, Config: core.Config{Project: "agent-board"}})
	if err != nil {
		return errors.New("cannot open offline registry")
	}
	defer r.Close()
	if operation == "token-revoke" {
		if err = r.RevokeCredential(context.Background(), *identity); err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(map[string]bool{"credential_revoked": true})
	}
	if operation == "recover-owner" {
		if *reason != "owner-recovery" || *project == "" {
			return errors.New("recover-owner requires project and --reason owner-recovery")
		}
		if err = r.RecoverOwner(context.Background(), *project, *identity); err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(map[string]bool{"owner_recovered": true})
	}
	if operation == "bootstrap-owner" {
		if err = r.BootstrapDefaultOwner(context.Background(), *identity); err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(map[string]any{"default_project": r.DefaultID(), "owner": *identity})
	}
	// Admin recovery and inventory are local and secret-free (docs/admin.md).
	if operation == "bootstrap-admin" {
		p, err := r.BootstrapAdmin(context.Background(), *identity)
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(map[string]any{"admin": p, "identity": *identity})
	}
	if operation == "people-list" {
		listing, err := r.People(context.Background())
		if err != nil {
			return err
		}
		return json.NewEncoder(out).Encode(listing)
	}
	if *days < 1 || *days > 365 || *output == "" {
		return errors.New("days 1..365 and new output required")
	}
	parent, err := os.Stat(filepath.Dir(*output))
	if err != nil || !parent.IsDir() || parent.Mode().Perm() != 0700 {
		return errors.New("output parent must be private mode0700")
	}
	file, err := os.OpenFile(*output, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return errors.New("choose new private output")
	}
	done := false
	defer func() {
		file.Close()
		if !done {
			os.Remove(*output)
		}
	}()
	token, who, err := r.CreateCredentialFor(context.Background(), *person, *owner, *machine, time.Duration(*days)*24*time.Hour)
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
		if revokeErr := r.RevokeCredential(context.Background(), who.ID); revokeErr != nil {
			return errors.New("token output and revocation failed; inspect offline registry before continuing")
		}
		return errors.New("token output failed; credential revoked")
	}
	done = true
	return json.NewEncoder(out).Encode(map[string]any{"identity": who, "expires_at": token.ExpiresAt})
}
