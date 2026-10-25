package projects

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

// adminHTTP serves the four POST endpoints of docs/admin.md. The CSRF and form
// checks already passed in catalogHTTP. Non-admins get 404 before any binding;
// payloads are validated before they become durable action state, and replay
// returns the stored result, which never contains a credential secret.
func (r *Registry) adminHTTP(w http.ResponseWriter, req *http.Request, b browserIdentity, parts []string) {
	if !b.Identity.Admin {
		fail(w, notFound())
		return
	}
	form := req.PostForm
	op, id, fields := "", "", []string{}
	switch {
	case len(parts) == 1 && parts[0] == "people":
		op, fields = "admin:person:create", []string{"name", "note", "role"}
	case len(parts) == 2 && parts[0] == "people" && validID(parts[1], "u_"):
		id = parts[1]
		op, fields = "admin:person:"+id+":update", []string{"action", "name", "note", "role", "version"}
	case len(parts) == 3 && parts[0] == "people" && validID(parts[1], "u_") && parts[2] == "credentials":
		id = parts[1]
		op, fields = "admin:person:"+id+":credential", []string{"machine", "days"}
	case len(parts) == 2 && parts[0] == "identities" && validIdentityID(parts[1]):
		id = parts[1]
		action := form.Get("action")
		if action != "revoke" && action != "link" {
			fail(w, invalid())
			return
		}
		op, fields = "admin:identity:"+id+":"+action, []string{"action", "person_id"}
	default:
		fail(w, notFound())
		return
	}
	allowed := map[string]bool{"csrf": true, "action_id": true}
	for _, key := range fields {
		allowed[key] = true
	}
	payload := map[string]string{}
	for key, values := range form {
		if !allowed[key] {
			fail(w, invalid())
			return
		}
		if key != "csrf" && key != "action_id" {
			payload[key] = values[0]
		}
	}
	if e := validateAdminPayload(op, payload); e != nil {
		fail(w, e)
		return
	}
	actionID := form.Get("action_id")
	cached, e := r.bindCatalog(req.Context(), b, actionID, op, "", payload, false)
	if e != nil {
		fail(w, e)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if cached != "" {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		w.Write([]byte(cached))
		return
	}
	ctx := context.WithValue(req.Context(), catalogActionKey{}, catalogAction{ID: actionID, Identity: b.Identity})
	var body map[string]any
	switch {
	case op == "admin:person:create":
		var p Person
		p, e = r.CreatePerson(ctx, b.Identity, payload["name"], payload["note"], payload["role"])
		body = map[string]any{"ok": true, "person": p}
	case fields[0] == "action" && parts[0] == "people":
		in := PersonUpdate{Action: payload["action"], Name: payload["name"], Note: payload["note"], Role: payload["role"]}
		if raw, ok := payload["version"]; ok && raw != "" {
			in.Version, _ = strconv.ParseInt(raw, 10, 64)
			in.VersionGiven = true
		}
		var p Person
		p, e = r.UpdatePerson(ctx, b.Identity, id, in)
		body = map[string]any{"ok": true, "person": p}
	case fields[0] == "machine":
		var issued IssuedCredential
		issued, e = r.IssuePersonCredential(ctx, b.Identity, id, payload["machine"], parseDays(payload["days"]))
		body = map[string]any{"ok": true, "issued": issued}
	case payload["action"] == "revoke":
		var view AdminIdentity
		view, e = r.RevokeIdentity(ctx, b.Identity, id)
		body = map[string]any{"ok": true, "identity": view}
	default:
		var view AdminIdentity
		view, e = r.LinkIdentity(ctx, b.Identity, id, payload["person_id"])
		body = map[string]any{"ok": true, "identity": view}
	}
	if e != nil {
		var replay *catalogReplay
		if errors.As(e, &replay) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.Write([]byte(replay.Result))
			return
		}
		fail(w, e)
		return
	}
	write(w, 200, body)
}

// validateAdminPayload rejects malformed fields before the action binds, so a
// durable payload is always one the mutation can accept.
func validateAdminPayload(op string, payload map[string]string) error {
	switch {
	case op == "admin:person:create":
		if !validPersonName(payload["name"]) || !validPersonNote(payload["note"]) || !validPersonRole(payload["role"]) {
			return invalid()
		}
	case strings.HasSuffix(op, ":update"):
		switch payload["action"] {
		case "update":
			if _, e := strconv.ParseInt(payload["version"], 10, 64); e != nil || !validPersonName(payload["name"]) || !validPersonNote(payload["note"]) || !validPersonRole(payload["role"]) {
				return invalid()
			}
		case "deactivate", "activate":
			if raw := payload["version"]; raw != "" {
				if _, e := strconv.ParseInt(raw, 10, 64); e != nil {
					return invalid()
				}
			}
		default:
			return invalid()
		}
	case strings.HasSuffix(op, ":credential"):
		if days := parseDays(payload["days"]); days < 1 || days > 365 || !validPersonName(payload["machine"]) {
			return invalid()
		}
	case strings.HasSuffix(op, ":link"):
		if !validID(payload["person_id"], "u_") {
			return invalid()
		}
	}
	return nil
}
