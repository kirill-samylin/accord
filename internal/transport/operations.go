// Package transport adapts the same authoritative service to HTTP and MCP.
package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	"agent-board/internal/core"
	"github.com/google/jsonschema-go/jsonschema"
)

// Service is the shared authenticated domain API; transports never accept a
// client-provided Principal or bypass its session and lease checks.
type Service interface {
	Profiles(context.Context, core.Actor, core.ProfilesInput) (core.ProfilesPage, error)
	Profile(context.Context, core.Actor, core.ProfileInput) (core.ProfilePage, error)
	ProfileCreate(context.Context, core.Actor, core.ProfileCreateInput) (core.AgentProfile, error)
	ProfileUpdate(context.Context, core.Actor, core.ProfileUpdateInput) (core.AgentProfile, error)
	ProfileArchive(context.Context, core.Actor, core.ProfileArchiveInput) (core.AgentProfile, error)

	Authenticate(context.Context, string) (core.Principal, error)
	Describe(context.Context, core.Principal, core.OpenSession) (core.SessionView, error)
	Next(context.Context, core.Actor, core.NextInput) (core.Claim, error)
	Get(context.Context, core.Actor, core.GetInput) (core.Task, error)
	Create(context.Context, core.Actor, core.CreateInput) (core.Task, error)
	Update(context.Context, core.Actor, core.UpdateInput) (core.Task, error)
	Transition(context.Context, core.Actor, core.TransitionInput) (core.Task, error)
	Release(context.Context, core.Actor, core.LeaseInput) (core.Task, error)
	Comment(context.Context, core.Actor, core.CommentInput) (core.Task, error)
	Link(context.Context, core.Actor, core.LinkInput) (core.Task, error)
	Heartbeat(context.Context, core.Actor, core.HeartbeatInput) (core.SessionView, error)
	Events(context.Context, core.Actor, core.EventsInput) (core.EventsPage, error)
	Note(context.Context, core.Actor, core.NoteInput) (core.Note, error)
	ListTasks(context.Context, core.Actor, core.ListInput) (core.TasksPage, error)
	ListEpics(context.Context, core.Actor, core.ListInput) (core.TasksPage, error)
	Sessions(context.Context, core.Actor, core.ListInput) (core.SessionsPage, error)
	ListNotes(context.Context, core.Actor, core.NotesInput) (core.NotesPage, error)
	Stats(context.Context, core.Actor, core.StatsInput) (core.Stats, error)
}

type operation struct {
	name, description string
	readOnly          bool
	schema            *jsonschema.Schema
	run               func(context.Context, core.Principal, string, []byte) (any, error)
}

// decode rejects unknown properties, null bodies, trailing values and malformed JSON.
func decode(data []byte, out any) error {
	if err := uniqueJSON(data); err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 || bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return core.Fail("VALIDATION", "JSON object is required")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	if err := d.Decode(out); err != nil {
		return core.Fail("VALIDATION", "invalid JSON fields or values")
	}
	if err := d.Decode(new(any)); !errors.Is(err, io.EOF) {
		return core.Fail("VALIDATION", "exactly one JSON object is required")
	}
	return nil
}

func bind[I, O any](name, description string, readOnly bool, call func(context.Context, core.Actor, I) (O, error)) operation {
	schema, err := jsonschema.ForType(reflect.TypeFor[I](), nil)
	if err != nil {
		panic("invalid internal operation schema")
	}
	return operation{name: name, description: description, readOnly: readOnly, schema: schema,
		run: func(ctx context.Context, p core.Principal, session string, data []byte) (any, error) {
			if session == "" {
				return nil, core.Fail("NO_SESSION", "board session is required")
			}
			var in I
			if err := decode(data, &in); err != nil {
				return nil, err
			}
			return call(ctx, core.Actor{Principal: p, SessionID: session}, in)
		}}
}

func operations(s Service) []operation {
	describe := bind("describe", "Open a board session with a fixed role and run ID.", false,
		func(ctx context.Context, a core.Actor, in core.OpenSession) (core.SessionView, error) {
			return s.Describe(ctx, a.Principal, in)
		})
	run := describe.run
	describe.run = func(ctx context.Context, p core.Principal, _ string, b []byte) (any, error) {
		return run(ctx, p, "bootstrap", b)
	}
	return []operation{
		bind("comment", "Record a short report on your current leased attempt.", false, s.Comment),
		bind("create", "Create an epic or a ready task.", false, s.Create), describe,
		bind("events", "Read one page; persist next_cursor, never high_water.", true, s.Events),
		bind("get", "Read an epic or task, including attempts and evidence.", true, s.Get),
		bind("heartbeat", "Renew only live leases using their exact generations.", false, s.Heartbeat),
		bind("link", "Add a typed dependency; cycles and unsuccessful gates are rejected.", false, s.Link),
		bind("next", "Claim one eligible task for this session's role.", false, s.Next),
		bind("note", "Create or soft-delete a bounded role or epic note with provenance.", false, s.Note),
		bind("release", "Release your live task lease with a reason.", false, s.Release),
		bind("transition", "Change state with current generation and required evidence.", false, s.Transition),
		bind("update", "Edit permitted metadata with ownership checks.", false, s.Update),
	}
}

// Reads exposed by REST and resources do not expand the twelve-tool MCP budget.
func readOperations(s Service) []operation {
	return []operation{
		bind("tasks", "List one stable task page.", true, s.ListTasks),
		bind("epics", "List one stable epic page.", true, s.ListEpics),
		bind("sessions", "List one session page.", true, s.Sessions),
		bind("notes", "Read one page of notes.", true, s.ListNotes),
		bind("stats", "Read historical totals with incomplete-data indicators.", true, s.Stats),
	}
}

// Profile CRUD is REST/CLI-only; profile resources preserve the MCP tool budget.
func profileOperations(s Service) []operation {
	ops := []operation{
		bind("profiles", "Read project profiles.", true, s.Profiles),
		bind("profile", "Read a profile or immutable revision.", true, s.Profile),
		bind("profile_create", "Create a persistent profile.", false, s.ProfileCreate),
		bind("profile_update", "Update a profile with version precondition.", false, s.ProfileUpdate),
		bind("profile_archive", "Archive or restore a profile.", false, s.ProfileArchive),
	}
	for i := range ops {
		if ops[i].readOnly {
			continue
		}
		run := ops[i].run
		name := ops[i].name
		ops[i].run = func(ctx context.Context, p core.Principal, session string, data []byte) (any, error) {
			var fields map[string]json.RawMessage
			if err := decodeObject(data, &fields); err != nil {
				return nil, err
			}
			required := []string{"request_id", "name", "description", "role", "instructions", "provider", "model"}
			if name == "profile_update" {
				required = append(required, "profile_id", "version")
			}
			if name == "profile_archive" {
				required = []string{"request_id", "profile_id", "version", "state"}
			}
			for _, key := range required {
				if raw, ok := fields[key]; !ok || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
					return nil, core.Fail("validation", "all profile fields are required")
				}
			}
			return run(ctx, p, session, data)
		}
	}
	return ops
}
