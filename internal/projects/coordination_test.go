package projects

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestCoordinationProjectACLForeignProfilesEpochAndArchive(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	a, b := f.create(t, f.who, "A"), f.create(t, f.who, "B")
	owner := f.describe(t, a.ID, f.token.Secret, "owner", "human")
	other := f.describe(t, b.ID, f.token.Secret, "other", "human")
	path := "/api/v1/p/" + a.ID
	post := func(route string, input any) []byte {
		raw, _ := json.Marshal(input)
		out := f.native(t, "POST", path+route, f.token.Secret, owner.SessionID, string(raw))
		if out.Code != 200 {
			t.Fatal(out.Code, out.Body.String())
		}
		return out.Body.Bytes()
	}
	var epic, task core.Task
	json.Unmarshal(post("/tasks", core.CreateInput{RequestID: "epic", Type: "epic", Title: "E"}), &epic)
	json.Unmarshal(post("/tasks", core.CreateInput{RequestID: "task", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Task", Description: "Work"}), &task)
	profileBody := `{"request_id":"profile","name":"Developer","description":"","role":"developer","instructions":"Pinned","provider":"unconfigured","model":""}`
	out := f.native(t, "POST", path+"/profile_create", f.token.Secret, owner.SessionID, profileBody)
	if out.Code != 200 {
		t.Fatal(out.Code)
	}
	var profile core.AgentProfile
	json.Unmarshal(out.Body.Bytes(), &profile)
	foreign := fmt.Sprintf(`{"request_id":"foreign","type":"task","epic_key":"E-1","role":"developer","title":"Foreign","description":"x","profile_id":%q,"profile_version":1}`, profile.ID)
	// Existing B epic gets its own opaque key; profile IDs never cross scope.
	bep := f.native(t, "POST", "/api/v1/p/"+b.ID+"/tasks", f.token.Secret, other.SessionID, `{"request_id":"epic","type":"epic","title":"B"}`)
	if bep.Code != 200 {
		t.Fatal(bep.Code)
	}
	if got := f.native(t, "POST", "/api/v1/p/"+b.ID+"/tasks", f.token.Secret, other.SessionID, foreign); got.Code != 404 {
		t.Fatal("foreign profile assignment", got.Code, got.Body.String())
	}
	cred, who, err := f.r.CreateCredential(ctx, "editor", "machine", time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	editor := f.describe(t, a.ID, cred.Secret, "editor", "human")
	body := fmt.Sprintf(`{"request_id":"assign","version":%d,"mode":"profile","profile_id":%q,"profile_version":1,"reason":"Route"}`, task.Version, profile.ID)
	route := path + "/tasks/" + task.Key
	if got := f.native(t, "POST", route+"/assign", cred.Secret, editor.SessionID, body); got.Code != 200 {
		t.Fatal("human editor assign", got.Code, got.Body.String())
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "viewer", "set", 1); err != nil {
		t.Fatal(err)
	}
	if got := f.native(t, "POST", route+"/assign", cred.Secret, editor.SessionID, body); got.Code != 403 {
		t.Fatal("viewer replay", got.Code)
	}
	if got := f.native(t, "GET", route+"/coordination", cred.Secret, editor.SessionID, ""); got.Code != 200 {
		t.Fatal("viewer read", got.Code)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "", "revoke", 2); err != nil {
		t.Fatal(err)
	}
	if err = f.r.changeMember(ctx, f.who, a.ID, who.ID, "editor", "grant", 0); err != nil {
		t.Fatal(err)
	}
	if got := f.native(t, "GET", route+"/coordination", cred.Secret, editor.SessionID, ""); got.Code != 401 {
		t.Fatal("old epoch revived", got.Code)
	}
	if _, err = f.r.updateProject(ctx, f.who, a.ID, a.Name, "archived", a.Version); err != nil {
		t.Fatal(err)
	}
	if got := f.native(t, "POST", route+"/assign", f.token.Secret, owner.SessionID, body); got.Code != 403 {
		t.Fatal("archived write", got.Code)
	}
	if got := f.native(t, "GET", route+"/coordination", f.token.Secret, owner.SessionID, ""); got.Code != 200 {
		t.Fatal("archive read", got.Code)
	}
}
