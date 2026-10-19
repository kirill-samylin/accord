// Package web implements browser sessions over the authoritative board API.
package web

import (
	"agent-board/internal/core"
	"context"
	"net/http"
	"time"
)

type Board interface {
	Authenticate(context.Context, string) (core.Principal, error)
	AuthenticateID(context.Context, string) (core.Principal, time.Time, error)
	Describe(context.Context, core.Principal, core.OpenSession) (core.SessionView, error)
	Next(context.Context, core.Actor, core.NextInput) (core.Claim, error)
	Get(context.Context, core.Actor, core.GetInput) (core.Task, error)
	Create(context.Context, core.Actor, core.CreateInput) (core.Task, error)
	Transition(context.Context, core.Actor, core.TransitionInput) (core.Task, error)
	Release(context.Context, core.Actor, core.LeaseInput) (core.Task, error)
	Heartbeat(context.Context, core.Actor, core.HeartbeatInput) (core.SessionView, error)
	Events(context.Context, core.Actor, core.EventsInput) (core.EventsPage, error)
	ListDecisions(context.Context, core.Actor, core.DecisionsInput) (core.DecisionsPage, error)
	ListQuestions(context.Context, core.Actor, core.ListInput) (core.TasksPage, error)
	ListTasks(context.Context, core.Actor, core.ListInput) (core.TasksPage, error)
	ListEpics(context.Context, core.Actor, core.ListInput) (core.TasksPage, error)
}

type Options struct {
	Origin, DataDir string
	Renderer        Renderer
	StaticHandler   http.Handler
	// TrustedProxy permits only a loopback peer's overwritten X-Accord-Client-IP.
	TrustedProxy bool
}
type Renderer interface {
	Render(http.ResponseWriter, int, string, Page) error
}

type Page struct {
	Title, Error, Notice, CSRF, ActionID, NextCursor      string
	EpicsNextCursor                                       string
	Principal                                             core.Principal
	Session                                               core.SessionView
	Task                                                  *core.Task
	Tasks, Epics, Questions, MyQuestions, LegacyDecisions []core.Task
	Designs                                               []DesignAsset
	Decisions                                             []core.Event
	CanContinue, CanEdit                                  bool
	Draft                                                 map[string]string
	Actions, FieldErrors                                  map[string]string
	View, EpicKey, Role, State                            string
	Roles, States, Categories, Complexities               []string
}

type DesignFilter struct {
	Scope, ScopeKey, DesignID, Cursor string
	Limit                             int
}
type DesignPage struct {
	Assets     []DesignAsset `json:"assets"`
	NextCursor string        `json:"next_cursor,omitempty"`
}
type DesignAsset struct {
	AssetID           string    `json:"asset_id"`
	DesignID          string    `json:"design_id"`
	Version           int       `json:"version"`
	Scope             string    `json:"scope"`
	ScopeKey          string    `json:"scope_key"`
	Title             string    `json:"title"`
	Filename          string    `json:"filename"`
	MediaType         string    `json:"media_type"`
	Size              int64     `json:"size"`
	SHA256            string    `json:"sha256"`
	UploaderPrincipal string    `json:"uploader_principal"`
	UploaderOwner     string    `json:"uploader_owner"`
	CreatedAt         time.Time `json:"created_at"`
	DownloadURL       string    `json:"download_url"`
	PreviewURL        string    `json:"preview_url,omitempty"`
}
