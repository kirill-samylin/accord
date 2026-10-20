package projects

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"strconv"
	"strings"
)

func catalogForm(w http.ResponseWriter, req *http.Request) error {
	if req.PostForm != nil {
		return nil
	}
	if !strings.HasPrefix(req.Header.Get("Content-Type"), "application/x-www-form-urlencoded") {
		return invalid()
	}
	req.Body = http.MaxBytesReader(w, req.Body, 64<<10)
	if req.ParseForm() != nil {
		return invalid()
	}
	for _, values := range req.PostForm {
		if len(values) != 1 {
			return invalid()
		}
	}
	return nil
}
func (r *Registry) catalogHTTP(w http.ResponseWriter, req *http.Request, b browserIdentity) {
	if req.Method == "GET" {
		r.catalogPage(w, req, b)
		return
	}
	if req.Method != "POST" {
		fail(w, invalid())
		return
	}
	if e := catalogForm(w, req); e != nil {
		fail(w, e)
		return
	}
	if !equal(req.PostForm.Get("csrf"), b.CSRF) {
		fail(w, denied())
		return
	}
	op, pid, rid := "", "", ""
	tail := strings.TrimPrefix(req.URL.Path, "/web/catalog/")
	parts := strings.Split(tail, "/")
	switch {
	case tail == "projects":
		op = "create"
	case tail == "switch":
		op = "switch"
		pid = req.PostForm.Get("project_id")
	case len(parts) >= 3 && parts[0] == "projects":
		pid = parts[1]
		switch parts[2] {
		case "update":
			if len(parts) == 3 {
				op = "update"
			}
		case "repositories":
			if len(parts) == 3 {
				op = "repository:add"
			} else if len(parts) == 4 {
				rid = parts[3]
				op = "repository:" + rid + ":" + req.PostForm.Get("action")
			}
		case "members":
			if len(parts) == 3 {
				op = "member:" + req.PostForm.Get("action")
				if req.PostForm.Get("action") != "grant" {
					op = "member:" + req.PostForm.Get("identity_id") + ":" + req.PostForm.Get("action")
				}
			}
		}
	}
	if op == "" {
		fail(w, notFound())
		return
	}
	allowed := map[string]bool{"csrf": true, "action_id": true}
	fields := []string{}
	switch {
	case op == "create":
		fields = []string{"name", "repository_url", "branch"}
	case op == "switch":
		fields = []string{"project_id", "target"}
	case op == "update":
		fields = []string{"name", "state", "version"}
	case strings.HasPrefix(op, "repository:"):
		fields = []string{"action", "url", "branch", "version"}
	case strings.HasPrefix(op, "member:"):
		fields = []string{"action", "identity_id", "role", "version"}
	}
	for _, key := range fields {
		allowed[key] = true
	}
	payload := map[string]string{}
	for key, values := range req.PostForm {
		if !allowed[key] {
			fail(w, invalid())
			return
		}
		if key != "csrf" && key != "action_id" {
			payload[key] = values[0]
		}
	}
	if strings.HasPrefix(op, "member:") {
		target := payload["identity_id"]
		if !validID(target, "i_") && !(strings.HasPrefix(target, "legacy_") && validHex(strings.TrimPrefix(target, "legacy_"), 64)) {
			fail(w, invalid())
			return
		}
	}
	if op == "create" && !validName(payload["name"]) {
		fail(w, invalid())
		return
	}
	if op == "create" && payload["repository_url"] != "" {
		url, branch, e := repositoryFields(payload["repository_url"], payload["branch"])
		if e != nil {
			fail(w, e)
			return
		}
		payload["repository_url"] = url
		payload["branch"] = branch
	}
	if strings.HasPrefix(op, "repository:") && !strings.HasSuffix(op, ":remove") {
		url, branch, e := repositoryFields(payload["url"], payload["branch"])
		if e != nil {
			fail(w, e)
			return
		}
		payload["url"] = url
		payload["branch"] = branch
	}
	actionID := req.PostForm.Get("action_id")
	wireOp := op
	if op == "switch" {
		wireOp = "switch:" + pid
	}
	cached, e := r.bindCatalog(req.Context(), b, actionID, wireOp, pid, payload, op != "create" && op != "switch")
	if e != nil {
		fail(w, e)
		return
	}
	if cached != "" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(cached))
		return
	}
	target := "/web/catalog/projects/" + pid
	if op == "create" {
		pid = "p_" + r.actionProjectID(actionID)
		target = "/web/catalog/projects/" + pid
	}
	if op == "switch" {
		target = payload["target"]
		if target == "" {
			target = "/web/p/" + pid + "/"
		}
	}
	result := actionResult(target)
	ctx := context.WithValue(req.Context(), catalogActionKey{}, catalogAction{ID: actionID, Result: result, Identity: b.Identity})
	version, _ := strconv.ParseInt(payload["version"], 10, 64)
	switch {
	case op == "create":
		_, e = r.createProject(ctx, b.Identity, pid, payload["name"], payload["repository_url"], payload["branch"])
	case op == "switch":
		target, e = r.switchProject(ctx, b, pid, target)
		if e == nil {
			_, e = transaction(r, ctx, func(tx *sql.Tx) (bool, error) {
				a, e := r.access(ctx, tx, b.Identity, pid, false)
				if e != nil {
					return false, e
				}
				_ = a
				return true, finishCatalog(ctx, tx)
			})
		}
	case op == "update":
		_, e = r.updateProject(ctx, b.Identity, pid, payload["name"], payload["state"], version)
	case strings.HasPrefix(op, "repository:"):
		operation := "add"
		if rid != "" {
			operation = payload["action"]
		}
		e = r.repository(ctx, b.Identity, pid, rid, operation, payload["url"], payload["branch"], version)
	case strings.HasPrefix(op, "member:"):
		e = r.changeMember(ctx, b.Identity, pid, payload["identity_id"], payload["role"], payload["action"], version)
	default:
		e = invalid()
	}
	if e != nil {
		var replay *catalogReplay
		if errors.As(e, &replay) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(replay.Result))
			return
		}
		fail(w, e)
		return
	}
	redirect(w, req, target)
}
func (r *Registry) catalogPage(w http.ResponseWriter, req *http.Request, b browserIdentity) {
	ctx := req.Context()
	registry := map[string]any{"identity": b.Identity, "csrf": b.CSRF}
	actions := map[string]string{}
	issue := func(key, pid string) error {
		id, e := r.issue(ctx, b, key, pid)
		if e == nil {
			actions[key] = id
		}
		return e
	}
	if e := issue("logout", ""); e != nil {
		fail(w, e)
		return
	}
	if req.URL.Path == "/web/projects" {
		projects, e := r.List(ctx, b.Identity)
		if e != nil {
			fail(w, e)
			return
		}
		registry["projects"] = projects
		if e = issue("create", ""); e != nil {
			fail(w, e)
			return
		}
		for _, project := range projects {
			if e = issue("switch:"+project.ID, project.ID); e != nil {
				fail(w, e)
				return
			}
		}
		registry["actions"] = actions
		write(w, 200, map[string]any{"ok": true, "page": map[string]any{"name": "projects", "registry": registry}})
		return
	}
	parts := strings.Split(strings.TrimPrefix(req.URL.Path, "/web/catalog/projects/"), "/")
	if len(parts) != 1 || !validID(parts[0], "p_") {
		fail(w, notFound())
		return
	}
	pid := parts[0]
	a, e := transaction(r, ctx, func(tx *sql.Tx) (access, error) { return r.access(ctx, tx, b.Identity, pid, false) })
	if e != nil {
		fail(w, e)
		return
	}
	page := map[string]any{"name": "project-settings", "registry": registry, "project": a.project}
	if e = issue("switch:"+pid, pid); e != nil {
		fail(w, e)
		return
	}
	if a.project.CanManage {
		members, e := r.members(ctx, b.Identity, pid)
		if e != nil {
			fail(w, e)
			return
		}
		page["members"] = members
		for _, key := range []string{"update", "repository:add", "member:grant"} {
			if e = issue(key, pid); e != nil {
				fail(w, e)
				return
			}
		}
		for _, repo := range a.project.Repositories {
			for _, op := range []string{"update", "remove"} {
				if e = issue("repository:"+repo.ID+":"+op, pid); e != nil {
					fail(w, e)
					return
				}
			}
		}
		for _, member := range members {
			for _, op := range []string{"set", "revoke"} {
				if e = issue("member:"+member.IdentityID+":"+op, pid); e != nil {
					fail(w, e)
					return
				}
			}
		}
	}
	registry["actions"] = actions
	write(w, 200, map[string]any{"ok": true, "page": page})
}
func (r *Registry) actionProjectID(action string) string {
	a := access{project: Project{ID: "catalog"}, identity: Identity{ID: "create"}}
	return r.describeID(a, action)[:32]
}
