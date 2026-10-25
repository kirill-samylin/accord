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
const MaxPersonName = 64
const MaxPersonNote = 512

type Repository = core.ProjectRepository
type Project = core.ProjectView
type Identity struct {
	ID      string `json:"id"`
	Owner   string `json:"owner"`
	Machine string `json:"machine"`
	// Admin is the global role of the identity's active person (docs/admin.md).
	Admin    bool      `json:"admin"`
	Source   string    `json:"-"`
	TokenID  string    `json:"-"`
	PersonID string    `json:"-"`
	Expires  time.Time `json:"-"`
}

// Person is the global account entity behind one or more identities.
type Person struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Note      string    `json:"note"`
	Role      string    `json:"role"`
	Active    bool      `json:"active"`
	Version   int64     `json:"version"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AdminIdentity is the secret-free credential view of the admin page and CLI.
type AdminIdentity struct {
	ID        string     `json:"id"`
	PersonID  string     `json:"person_id"`
	Owner     string     `json:"owner"`
	Machine   string     `json:"machine"`
	Source    string     `json:"source"`
	ExpiresAt *time.Time `json:"expires_at"`
	Revoked   bool       `json:"revoked"`
}
type AdminAudit struct {
	ID         int64     `json:"id"`
	Event      string    `json:"event"`
	ActorID    string    `json:"actor_id"`
	PersonID   string    `json:"person_id"`
	IdentityID string    `json:"identity_id"`
	Details    string    `json:"details"`
	At         time.Time `json:"at"`
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
	// adminOnly marks a global admin without membership: catalog member
	// management only (docs/admin.md), no board, archive or repositories.
	adminOnly bool
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
