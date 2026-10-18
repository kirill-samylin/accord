package store

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"slices"

	"time"

	"agent-board/internal/config"
	"agent-board/internal/core"
)

var legacyRoles = []string{"coordinator", "analyst", "architect", "developer", "reviewer", "tester", "human"}

// bootstrapCompatibility preserves the pilot's real startup configuration. The
// Davinci profile is never implicitly applied to an existing project.
func bootstrapCompatibility(ctx context.Context, tx *sql.Tx, cfg core.Config) (*config.Snapshot, error) {
	c := config.Config{}
	raw := bytes.ReplaceAll(config.DavinciTemplate(), []byte("OWNER"), []byte("bootstrap-owner"))
	initial, result := config.Parse(raw, config.Options{})
	if !result.Valid {
		return nil, core.Fail("CONFIG_INTERNAL", "invalid embedded compatibility source")
	}
	c = initial.Decode()
	c.Project.Name = cfg.Project
	if c.Project.Name == "" {
		c.Project.Name = "agent-board"
	}
	c.Leases.TaskTTL = cfg.LeaseTTL.String()
	c.Leases.RoleTTL = cfg.SessionTTL.String()
	c.Leases.SessionTTL = cfg.SessionTTL.String()
	c.Leases.MaxSilent = min(cfg.LeaseTTL, cfg.SessionTTL).String()
	c.Leases.SweepInterval = min(30*time.Second, cfg.LeaseTTL, cfg.SessionTTL).String()
	c.Auth.Local.Users = append([]string{}, cfg.BootstrapUsers...)
	rows, err := tx.QueryContext(ctx, `SELECT DISTINCT owner FROM tokens ORDER BY owner`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var owner string
		if err = rows.Scan(&owner); err != nil {
			rows.Close()
			return nil, err
		}
		if !slices.Contains(c.Auth.Local.Users, owner) {
			c.Auth.Local.Users = append(c.Auth.Local.Users, owner)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	if len(c.Auth.Local.Users) == 0 {
		return nil, core.Fail("CONFIG_BOOTSTRAP_REQUIRED", "new database requires explicit bootstrap owner")
	}
	for role := range c.Roles {
		if !slices.Contains(legacyRoles, role) {
			delete(c.Roles, role)
		}
	}
	for name, role := range c.Roles {
		role.EvidenceOnDone = config.RequiredFloor(name)
		role.RecommendedOnDone = []string{}
		role.Members = []string{}
		role.Categories = map[string]config.Category{}
		if name == "human" {
			role.CanCreate = []string{"task", "epic"}
			role.CanCancel = true
		}
		c.Roles[name] = role
	}
	c.Reopen = map[string][]string{"coordinator": {"*"}, "human": {"*"}, "reviewer": {"developer"}, "tester": {"developer"}, "developer": {"analyst", "architect"}}
	c.MCP.ToolsByRole = map[string][]string{}
	c.Rules.EpicOwner.Mode = "off"
	c.Rules.EpicOwner.SharedRoles = []string{"reviewer", "tester"}
	c.Rules.AreaOverlap.Mode = "off"
	c.Rules.DuplicateCheck.Mode = "off"
	limits := cfg.Notes
	defaultNoteLimits(&limits)
	c.Notes = config.Notes{MaxPerRole: limits.MaxPerRole, MaxBytesPerRole: limits.MaxBytesPerRole, MaxPerEpic: limits.MaxPerEpic, MaxBytesPerEpic: limits.MaxBytesPerEpic}
	encoded, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	snapshot, validation := config.Parse(encoded, config.Options{})
	if !validation.Valid {
		return nil, core.Fail("CONFIG_INVALID", "bootstrap configuration invalid: "+validation.Errors[0].Path+": "+validation.Errors[0].Message)
	}
	return snapshot, nil
}

func runtimeConfig(snapshot *config.Snapshot, version int64) core.Config {
	c := snapshot.Decode()
	task, role, session, _ := snapshot.TTLs()
	return core.Config{SessionTTL: session, RoleTTL: role, LeaseTTL: task, Project: c.Project.Name, ConfigVersion: version, Snapshot: snapshot, Workflow: c, Notes: core.NoteLimits{MaxPerRole: c.Notes.MaxPerRole, MaxBytesPerRole: c.Notes.MaxBytesPerRole, MaxPerEpic: c.Notes.MaxPerEpic, MaxBytesPerEpic: c.Notes.MaxBytesPerEpic}}
}

func loadActiveConfig(ctx context.Context, tx *sql.Tx) (core.Config, error) {
	var version int64
	var raw []byte
	err := tx.QueryRowContext(ctx, `SELECT h.version,h.canonical_json FROM config_history h JOIN config_active a ON a.version=h.version WHERE a.id=1`).Scan(&version, &raw)
	if err != nil {
		return core.Config{}, err
	}
	snapshot, result := config.Parse(raw, config.Options{})
	if !result.Valid {
		return core.Config{}, core.Fail("CONFIG_INTERNAL", "persisted configuration is invalid")
	}
	return runtimeConfig(snapshot, version), nil
}
