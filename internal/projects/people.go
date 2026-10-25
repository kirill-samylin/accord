package projects

import (
	"agent-board/internal/core"
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

// People are the global accounts of docs/admin.md. Every mutation here runs in
// one registry transaction, requires an active admin actor (otherwise 404, the
// page does not exist for members) and appends admin_audit without secrets.
func validPersonName(name string) bool {
	n := utf8.RuneCountInString(name)
	return core.PlainText(name, false) && strings.TrimSpace(name) == name && n >= 1 && n <= MaxPersonName
}
func validPersonNote(note string) bool {
	return core.PlainText(note, true) && utf8.RuneCountInString(note) <= MaxPersonNote
}
func validPersonRole(role string) bool { return role == "member" || role == "admin" }
func validIdentityID(id string) bool {
	return validID(id, "i_") || strings.HasPrefix(id, "legacy_") && validHex(strings.TrimPrefix(id, "legacy_"), 64)
}
func adminAudit(ctx context.Context, tx *sql.Tx, event, actor, person, identity, details string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO admin_audit(event,actor_id,person_id,identity_id,details,at) VALUES(?,?,?,?,?,?)`, event, actor, person, identity, details, time.Now().UnixNano())
	return err
}
func (r *Registry) requireAdmin(ctx context.Context, tx *sql.Tx, who Identity) (Identity, error) {
	current, err := r.identity(ctx, tx, who.ID)
	if err != nil {
		return current, err
	}
	if !current.Admin {
		return current, notFound()
	}
	return current, nil
}
func scanPerson(row interface{ Scan(...any) error }) (Person, error) {
	var p Person
	var created, updated int64
	err := row.Scan(&p.ID, &p.Name, &p.Note, &p.Role, &p.Active, &p.Version, &created, &updated)
	p.CreatedAt = time.Unix(0, created).UTC()
	p.UpdatedAt = time.Unix(0, updated).UTC()
	return p, err
}
func (r *Registry) person(ctx context.Context, tx *sql.Tx, id string) (Person, error) {
	if !validID(id, "u_") {
		return Person{}, notFound()
	}
	p, err := scanPerson(tx.QueryRowContext(ctx, `SELECT id,name,note,role,active,version,created,updated FROM people WHERE id=?`, id))
	if missing(err) {
		return p, notFound()
	}
	return p, err
}
func (r *Registry) createPerson(ctx context.Context, tx *sql.Tx, name, note, role string) (Person, error) {
	if !validPersonName(name) || !validPersonNote(note) || !validPersonRole(role) {
		return Person{}, invalid()
	}
	id, err := random("u_", 16)
	if err != nil {
		return Person{}, err
	}
	now := time.Now().UnixNano()
	_, err = tx.ExecContext(ctx, `INSERT INTO people(id,name,note,role,active,version,created,updated) VALUES(?,?,?,?,1,1,?,?)`, id, name, note, role, now, now)
	if err != nil {
		return Person{}, err
	}
	return r.person(ctx, tx, id)
}

// ensurePerson gives an identity created outside the admin page its own
// member person. Labels are never merged automatically.
func (r *Registry) ensurePerson(ctx context.Context, tx *sql.Tx, identityID, owner string) error {
	var personID string
	if err := tx.QueryRowContext(ctx, `SELECT person_id FROM identities WHERE id=?`, identityID).Scan(&personID); err != nil || personID != "" {
		return err
	}
	name := truncateRunes(strings.TrimSpace(owner), MaxPersonName)
	if !validPersonName(name) {
		name = "owner"
	}
	p, err := r.createPerson(ctx, tx, name, "", "member")
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE identities SET person_id=? WHERE id=?`, p.ID, identityID)
	return err
}
func (r *Registry) requireAnotherAdmin(ctx context.Context, tx *sql.Tx) error {
	var admins int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM people WHERE role='admin' AND active=1`).Scan(&admins); err != nil {
		return err
	}
	if admins <= 1 {
		return core.Fail("conflict", "last active admin must remain")
	}
	return nil
}

// requireAdminCredentialBesides refuses an operation that would leave no
// usable credential of an active admin: the identity being revoked or moved
// away is excluded from the count. Legacy tokens count as usable here; their
// revocation state lives on the default board and is checked when used.
func (r *Registry) requireAdminCredentialBesides(ctx context.Context, tx *sql.Tx, identityID string) error {
	var usable int
	err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM identities i JOIN people p ON p.id=i.person_id
		LEFT JOIN credentials c ON c.identity_id=i.id
		WHERE p.role='admin' AND p.active=1 AND i.id<>? AND
		(i.source='legacy' OR c.revoked=0 AND c.expires>?)`, identityID, time.Now().UnixNano()).Scan(&usable)
	if err != nil {
		return err
	}
	if usable == 0 {
		return core.Fail("conflict", "last admin credential must remain")
	}
	return nil
}
func personResult(ctx context.Context, tx *sql.Tx, p Person) error {
	raw, err := json.Marshal(map[string]any{"ok": true, "person": p})
	if err != nil {
		return err
	}
	return finishCatalogResult(ctx, tx, string(raw))
}
func (r *Registry) CreatePerson(ctx context.Context, actor Identity, name, note, role string) (Person, error) {
	if !validPersonName(name) || !validPersonNote(note) || !validPersonRole(role) {
		return Person{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (Person, error) {
		admin, err := r.requireAdmin(ctx, tx, actor)
		if err != nil {
			return Person{}, err
		}
		p, err := r.createPerson(ctx, tx, name, note, role)
		if err != nil {
			return p, err
		}
		if err = adminAudit(ctx, tx, "person_created", admin.ID, p.ID, "", "role="+role); err != nil {
			return p, err
		}
		return p, personResult(ctx, tx, p)
	})
}

// PersonUpdate carries the update|deactivate|activate form. Version is checked
// whenever the form sends one and is mandatory for update (docs/admin.md).
type PersonUpdate struct {
	Action, Name, Note, Role string
	Version                  int64
	VersionGiven             bool
}

func (r *Registry) UpdatePerson(ctx context.Context, actor Identity, id string, in PersonUpdate) (Person, error) {
	switch in.Action {
	case "update":
		if !in.VersionGiven || !validPersonName(in.Name) || !validPersonNote(in.Note) || !validPersonRole(in.Role) {
			return Person{}, invalid()
		}
	case "deactivate", "activate":
	default:
		return Person{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (Person, error) {
		admin, err := r.requireAdmin(ctx, tx, actor)
		if err != nil {
			return Person{}, err
		}
		p, err := r.person(ctx, tx, id)
		if err != nil {
			return p, err
		}
		if in.VersionGiven && in.Version != p.Version {
			return p, core.Fail("conflict", "person changed")
		}
		if in.Action == "deactivate" && !p.Active || in.Action == "activate" && p.Active {
			return p, personResult(ctx, tx, p)
		}
		switch in.Action {
		case "update":
			if p.Active && p.Role == "admin" && in.Role != "admin" {
				if err = r.requireAnotherAdmin(ctx, tx); err != nil {
					return p, err
				}
			}
			_, err = tx.ExecContext(ctx, `UPDATE people SET name=?,note=?,role=?,version=version+1,updated=? WHERE id=?`, in.Name, in.Note, in.Role, time.Now().UnixNano(), id)
		case "deactivate":
			if admin.PersonID == id {
				return p, core.Fail("forbidden", "cannot deactivate yourself")
			}
			if p.Active && p.Role == "admin" {
				if err = r.requireAnotherAdmin(ctx, tx); err != nil {
					return p, err
				}
			}
			_, err = tx.ExecContext(ctx, `UPDATE people SET active=0,version=version+1,updated=? WHERE id=?`, time.Now().UnixNano(), id)
		case "activate":
			_, err = tx.ExecContext(ctx, `UPDATE people SET active=1,version=version+1,updated=? WHERE id=?`, time.Now().UnixNano(), id)
		}
		if err != nil {
			return p, err
		}
		if p, err = r.person(ctx, tx, id); err != nil {
			return p, err
		}
		details := ""
		if in.Action == "update" {
			details = "role=" + in.Role
		}
		if err = adminAudit(ctx, tx, "person_"+in.Action+"d", admin.ID, id, "", details); err != nil {
			return p, err
		}
		return p, personResult(ctx, tx, p)
	})
}

// BootstrapAdmin is the offline recovery path: the identity's person (created
// when absent) becomes an active admin. The identity ID may be a registry ID
// or a bare legacy token ID, which is imported first.
func (r *Registry) BootstrapAdmin(ctx context.Context, identityID string) (Person, error) {
	if validHex(identityID, 64) {
		p, until, err := r.options.Default.AuthenticateID(ctx, identityID)
		if err != nil {
			return Person{}, err
		}
		if _, err = r.importLegacy(ctx, p, until); err != nil {
			return Person{}, err
		}
		identityID = "legacy_" + identityID
	}
	if !validIdentityID(identityID) {
		return Person{}, invalid()
	}
	return transaction(r, ctx, func(tx *sql.Tx) (Person, error) {
		var owner, personID string
		err := tx.QueryRowContext(ctx, `SELECT owner,person_id FROM identities WHERE id=?`, identityID).Scan(&owner, &personID)
		if missing(err) {
			return Person{}, notFound()
		}
		if err != nil {
			return Person{}, err
		}
		if err = r.ensurePerson(ctx, tx, identityID, owner); err != nil {
			return Person{}, err
		}
		if err = tx.QueryRowContext(ctx, `SELECT person_id FROM identities WHERE id=?`, identityID).Scan(&personID); err != nil {
			return Person{}, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE people SET role='admin',active=1,version=version+1,updated=? WHERE id=?`, time.Now().UnixNano(), personID); err != nil {
			return Person{}, err
		}
		if err = adminAudit(ctx, tx, "admin_bootstrapped", "cli", personID, identityID, ""); err != nil {
			return Person{}, err
		}
		return r.person(ctx, tx, personID)
	})
}
