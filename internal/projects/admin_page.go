package projects

import (
	"context"
	"database/sql"
	"net/http"
	"time"
)

type adminProject struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	State string `json:"state"`
}
type adminMembership struct {
	ProjectID  string `json:"project_id"`
	IdentityID string `json:"identity_id"`
	Role       string `json:"role"`
}

// AdminListing is the secret-free inventory of the admin page and the offline
// people-list command: people, identities, projects, memberships and audit.
type AdminListing struct {
	People      []Person          `json:"people"`
	Identities  []AdminIdentity   `json:"identities"`
	Projects    []adminProject    `json:"projects"`
	Memberships []adminMembership `json:"memberships"`
	Audit       []AdminAudit      `json:"audit"`
}

func (r *Registry) adminList(ctx context.Context, tx *sql.Tx) (AdminListing, error) {
	out := AdminListing{People: []Person{}, Identities: []AdminIdentity{}, Projects: []adminProject{}, Memberships: []adminMembership{}, Audit: []AdminAudit{}}
	rows, err := tx.QueryContext(ctx, `SELECT id,name,note,role,active,version,created,updated FROM people ORDER BY created,id`)
	if err != nil {
		return out, err
	}
	for rows.Next() {
		p, e := scanPerson(rows)
		if e != nil {
			rows.Close()
			return out, e
		}
		out.People = append(out.People, p)
	}
	if err = closeRows(rows); err != nil {
		return out, err
	}
	ids := []string{}
	if rows, err = tx.QueryContext(ctx, `SELECT id FROM identities ORDER BY id`); err != nil {
		return out, err
	}
	for rows.Next() {
		var id string
		if e := rows.Scan(&id); e != nil {
			rows.Close()
			return out, e
		}
		ids = append(ids, id)
	}
	if err = closeRows(rows); err != nil {
		return out, err
	}
	tokens := r.legacyTokens(ctx)
	for _, id := range ids {
		view, e := r.adminIdentity(ctx, tx, id, tokens)
		if e != nil {
			return out, e
		}
		out.Identities = append(out.Identities, view)
	}
	if rows, err = tx.QueryContext(ctx, `SELECT id,name,state FROM projects WHERE state<>'provisioning' ORDER BY created,id`); err != nil {
		return out, err
	}
	for rows.Next() {
		var p adminProject
		if e := rows.Scan(&p.ID, &p.Name, &p.State); e != nil {
			rows.Close()
			return out, e
		}
		out.Projects = append(out.Projects, p)
	}
	if err = closeRows(rows); err != nil {
		return out, err
	}
	if rows, err = tx.QueryContext(ctx, `SELECT project_id,identity_id,role FROM memberships WHERE active=1 ORDER BY project_id,identity_id`); err != nil {
		return out, err
	}
	for rows.Next() {
		var m adminMembership
		if e := rows.Scan(&m.ProjectID, &m.IdentityID, &m.Role); e != nil {
			rows.Close()
			return out, e
		}
		out.Memberships = append(out.Memberships, m)
	}
	if err = closeRows(rows); err != nil {
		return out, err
	}
	if rows, err = tx.QueryContext(ctx, `SELECT id,event,actor_id,person_id,identity_id,details,at FROM admin_audit ORDER BY id DESC LIMIT 200`); err != nil {
		return out, err
	}
	for rows.Next() {
		var a AdminAudit
		var at int64
		if e := rows.Scan(&a.ID, &a.Event, &a.ActorID, &a.PersonID, &a.IdentityID, &a.Details, &at); e != nil {
			rows.Close()
			return out, e
		}
		a.At = time.Unix(0, at).UTC()
		out.Audit = append(out.Audit, a)
	}
	return out, closeRows(rows)
}
func closeRows(rows *sql.Rows) error {
	err := rows.Err()
	if e := rows.Close(); err == nil {
		err = e
	}
	return err
}

// People is the offline people-list command: it never returns secrets.
func (r *Registry) People(ctx context.Context) (AdminListing, error) {
	return transaction(r, ctx, func(tx *sql.Tx) (AdminListing, error) { return r.adminList(ctx, tx) })
}

// adminPage serves GET /web/catalog/admin for admins only; others see 404.
func (r *Registry) adminPage(w http.ResponseWriter, req *http.Request, b browserIdentity, registry map[string]any, issue func(key, pid string) error) {
	ctx := req.Context()
	if !b.Identity.Admin {
		fail(w, notFound())
		return
	}
	listing, err := transaction(r, ctx, func(tx *sql.Tx) (AdminListing, error) {
		if _, e := r.requireAdmin(ctx, tx, b.Identity); e != nil {
			return AdminListing{}, e
		}
		return r.adminList(ctx, tx)
	})
	if err != nil {
		fail(w, err)
		return
	}
	keys := []string{"admin:person:create"}
	for _, p := range listing.People {
		keys = append(keys, "admin:person:"+p.ID+":update", "admin:person:"+p.ID+":credential")
	}
	for _, i := range listing.Identities {
		keys = append(keys, "admin:identity:"+i.ID+":link")
		if !i.Revoked {
			keys = append(keys, "admin:identity:"+i.ID+":revoke")
		}
	}
	for _, key := range keys {
		if err = issue(key, ""); err != nil {
			fail(w, err)
			return
		}
	}
	write(w, 200, map[string]any{"ok": true, "page": map[string]any{"name": "admin", "registry": registry, "people": listing.People, "identities": listing.Identities, "projects": listing.Projects, "memberships": listing.Memberships, "audit": listing.Audit}})
}
