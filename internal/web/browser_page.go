package web

import (
	"agent-board/internal/core"
	"time"
)

// This allowlist is intentionally separate from browserSession and persisted
// actions: browser hashes, bearer/token IDs, and pending payloads are private.
type browserPage struct {
	Coordination       *core.TaskCoordination `json:"coordination,omitempty"`
	AssignmentProfiles []core.AgentProfile    `json:"assignment_profiles,omitempty"`
	CanCoordinate      bool                   `json:"can_coordinate"`
	Profiles           []core.AgentProfile    `json:"profiles,omitempty"`
	Profile            *core.AgentProfile     `json:"profile,omitempty"`
	ProfileRevisions   []core.ProfileRevision `json:"profile_revisions,omitempty"`
	CanManageProfiles  bool                   `json:"can_manage_profiles"`
	Project            *core.ProjectView      `json:"project,omitempty"`
	Name               string                 `json:"name"`
	Title              string                 `json:"title"`
	View               string                 `json:"view"`
	Notice             string                 `json:"notice,omitempty"`
	CSRF               string                 `json:"csrf,omitempty"`
	ActionID           string                 `json:"action_id,omitempty"`
	Actions            map[string]string      `json:"actions,omitempty"`
	Draft              map[string]string      `json:"draft,omitempty"`
	FieldErrors        map[string]string      `json:"field_errors,omitempty"`
	Principal          browserPrincipal       `json:"principal"`
	Session            core.SessionView       `json:"session"`
	CanContinue        bool                   `json:"can_continue"`
	CanEdit            bool                   `json:"can_edit"`
	CanUpdate          bool                   `json:"can_update"`
	Overview           *core.ControlOverview  `json:"overview,omitempty"`
	Agents             []core.ControlSession  `json:"agents,omitempty"`
	Agent              *core.ControlSession   `json:"agent,omitempty"`
	Activity           []core.Event           `json:"activity,omitempty"`
	Stats              *core.Stats            `json:"stats,omitempty"`
	ObservedAt         time.Time              `json:"observed_at"`
	AgentsObservedAt   time.Time              `json:"agents_observed_at"`
	ActivityObservedAt time.Time              `json:"activity_observed_at"`
	Task               *core.Task             `json:"task,omitempty"`
	Tasks              []core.Task            `json:"tasks,omitempty"`
	Epics              []core.Task            `json:"epics,omitempty"`
	Questions          []core.Task            `json:"questions,omitempty"`
	MyQuestions        []core.Task            `json:"my_questions,omitempty"`
	LegacyDecisions    []core.Task            `json:"legacy_decisions,omitempty"`
	Designs            []DesignAsset          `json:"designs,omitempty"`
	Decisions          []core.Event           `json:"decisions,omitempty"`
	NextCursor         string                 `json:"next_cursor,omitempty"`
	EpicsNextCursor    string                 `json:"epics_next_cursor,omitempty"`
	Query              string                 `json:"query,omitempty"`
	AgentState         string                 `json:"agent_state,omitempty"`
	EpicKey            string                 `json:"epic_key,omitempty"`
	Role               string                 `json:"role,omitempty"`
	State              string                 `json:"state,omitempty"`
	Roles              []string               `json:"roles,omitempty"`
	States             []string               `json:"states,omitempty"`
	Categories         []string               `json:"categories,omitempty"`
	Complexities       []string               `json:"complexities,omitempty"`
}
type browserPrincipal struct {
	Owner   string `json:"owner"`
	Machine string `json:"machine"`
}

func projectBrowserPage(name string, p Page) browserPage {
	draft := make(map[string]string, len(p.Draft))
	for key, value := range p.Draft {
		if key != "token" && key != "token_file" && key != "csrf" {
			draft[key] = value
		}
	}
	return browserPage{
		Coordination: p.Coordination, AssignmentProfiles: p.AssignmentProfiles, CanCoordinate: p.CanCoordinate,
		Profiles: p.Profiles, Profile: p.Profile, ProfileRevisions: p.ProfileRevisions, CanManageProfiles: p.CanManageProfiles,
		Project: p.Project, Name: name, Title: p.Title, View: p.View, Notice: p.Notice, CSRF: p.CSRF,
		ActionID: p.ActionID, Actions: p.Actions, Draft: draft, FieldErrors: p.FieldErrors,
		Principal: browserPrincipal{Owner: p.Principal.Owner, Machine: p.Principal.Machine},
		Session:   p.Session, CanContinue: p.CanContinue, CanEdit: p.CanEdit, CanUpdate: p.CanUpdate,
		Overview: p.Overview, Agents: p.Agents, Agent: p.Agent, Activity: p.Activity, Stats: p.Stats,
		ObservedAt: p.ObservedAt, AgentsObservedAt: p.AgentsObservedAt, ActivityObservedAt: p.ActivityObservedAt,
		Task: p.Task, Tasks: p.Tasks, Epics: p.Epics, Questions: p.Questions, MyQuestions: p.MyQuestions,
		LegacyDecisions: p.LegacyDecisions, Designs: p.Designs, Decisions: p.Decisions,
		NextCursor: p.NextCursor, EpicsNextCursor: p.EpicsNextCursor, Query: p.Query,
		AgentState: p.AgentState, EpicKey: p.EpicKey, Role: p.Role, State: p.State,
		Roles: p.Roles, States: p.States, Categories: p.Categories, Complexities: p.Complexities,
	}
}
