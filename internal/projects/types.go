// Package projects owns project discovery, identity and isolated board routing.
package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/web"
	"context"
	"database/sql"
	"net/http"
	"sync"
	"time"
)

const MaxProjects = 32
const MaxOwnedProjects = 8
const MaxRepositories = 16
const MaxMembers = 128
const MaxProvisions = 2

type Repository = core.ProjectRepository
type Project = core.ProjectView
type Identity struct {
	ID      string    `json:"id"`
	Owner   string    `json:"owner"`
	Machine string    `json:"machine"`
	Source  string    `json:"-"`
	TokenID string    `json:"-"`
	Expires time.Time `json:"-"`
}
type Member struct {
	IdentityID string `json:"identity_id"`
	Owner      string `json:"owner"`
	Machine    string `json:"machine"`
	Role       string `json:"role"`
	Version    int64  `json:"version"`
}
type member struct {
	Member
	LocalID string
	Epoch   int64
	Active  bool
}
type Options struct {
	DataDir, DefaultDB, DefaultWeb, Origin string
	Default                                *store.Store
	Browser                                *web.Server
	Config                                 core.Config
	TrustedProxy                           bool
	Offline                                bool
}
type Registry struct {
	db         *sql.DB
	options    Options
	defaultID  string
	signingKey []byte
	mu         sync.Mutex
	runtimes   map[string]*runtime
}
type runtime struct {
	store   *store.Store
	browser *web.Server
	service *service
	native  http.Handler
}
type access struct {
	tx        *sql.Tx
	identity  Identity
	project   Project
	member    member
	principal core.Principal
}
type accountKey struct{}

func accountContext(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, accountKey{}, identity)
}
func accountFrom(ctx context.Context) (Identity, error) {
	value, ok := ctx.Value(accountKey{}).(Identity)
	if !ok {
		return Identity{}, core.Fail("unauthorized", "authentication required")
	}
	return value, nil
}
