package transport

import (
	"agent-board/internal/core"
	"context"
	"encoding/json"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/url"
	"testing"
	"time"
)

func TestCoordinationRESTStrictRoutingAndMCPResource(t *testing.T) {
	f := newFixture(t)
	session := openSession(t, f, "human", "coord")
	post := func(path string, input any) core.Task {
		raw, _ := json.Marshal(input)
		status, data := request(t, f, "POST", path, session.SessionID, string(raw))
		return parseResult[core.Task](t, status, data)
	}
	epic := post("/api/v1/tasks", core.CreateInput{RequestID: "epic", Type: "epic", Title: "Coordination"})
	code, raw := request(t, f, "POST", "/api/v1/profile_create", session.SessionID, `{"request_id":"profile","name":"Developer","description":"","role":"developer","instructions":"Pinned","provider":"unconfigured","model":""}`)
	profile := parseResult[core.AgentProfile](t, code, raw)
	task := post("/api/v1/tasks", core.CreateInput{RequestID: "task", Type: "task", EpicKey: epic.Key, Role: "developer", Title: "Task", Description: "Work", ProfileID: profile.ID, ProfileVersion: 1})
	path := "/api/v1/tasks/" + task.Key
	for _, body := range []string{`{"request_id":"a","version":1,"mode":"role_queue","reason":"x","profile_id":null}`, `{"request_id":"a","version":1,"mode":"role_queue","reason":"x","mode":"profile"}`, `{"request_id":"a","version":1,"mode":"role_queue","reason":"x","command":"bad"}`, `{"request_id":"a","task_key":"foreign","version":1,"mode":"role_queue","reason":"x"}`} {
		if status, _ := request(t, f, "POST", path+"/assign", session.SessionID, body); status != 400 {
			t.Fatal("invalid assignment accepted", status, body)
		}
	}
	for _, query := range []string{"?task_key=" + task.Key, "?unknown=1", "?x=1&x=2"} {
		if status, _ := request(t, f, "GET", path+"/coordination"+query, session.SessionID, ""); status != 400 {
			t.Fatal("query accepted")
		}
	}
	for _, suffix := range []string{"?unknown=1", "?%zz=x"} {
		for _, op := range []string{"assign", "checks"} {
			if status, _ := request(t, f, "POST", path+"/"+op+suffix, session.SessionID, `{}`); status != 400 {
				t.Fatal("mutation query accepted", status)
			}
		}
	}
	clear := post(path+"/assign", core.AssignInput{RequestID: "assign", TaskKey: task.Key, Version: task.Version, Mode: "role_queue", Reason: "General queue"})
	if clear.Assignment != nil {
		t.Fatal("clear failed")
	}
	if status, _ := request(t, f, "POST", path+"/checks", session.SessionID, `{"request_id":"c","target_key":"other","target_version":1,"role":"reviewer"}`); status != 400 {
		t.Fatal("checks route mismatch")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	consumer := mcp.NewClient(&mcp.Implementation{Name: "coordination", Version: "1"}, nil)
	conn, err := consumer.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.server.URL + "/mcp", HTTPClient: &http.Client{Transport: authenticatedTransport{f.token}}, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	tools, err := conn.ListTools(ctx, nil)
	if err != nil || len(tools.Tools) != 12 {
		t.Fatal("tool budget")
	}
	uri := "accord://coordination?" + url.Values{"session_id": {session.SessionID}, "task_key": {task.Key}}.Encode()
	result, err := conn.ReadResource(ctx, &mcp.ReadResourceParams{URI: uri})
	if err != nil || len(result.Contents) != 1 {
		t.Fatal(err)
	}
	var out core.TaskCoordination
	if err = json.Unmarshal([]byte(result.Contents[0].Text), &out); err != nil || out.TaskKey != task.Key || out.TaskVersion != clear.Version {
		t.Fatal("resource differs", err)
	}
	for _, bad := range []string{uri + "&task_key=other", uri + "&unknown=1", "accord://coordination?task_key=" + task.Key} {
		if _, err = conn.ReadResource(ctx, &mcp.ReadResourceParams{URI: bad}); err == nil {
			t.Fatal("invalid coordination resource accepted")
		}
	}
}
