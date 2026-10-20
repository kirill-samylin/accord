package main

import (
	"encoding/json"
	"testing"
)

func TestCoordinationClientRoutesPinPathAndRetainMutationPayload(t *testing.T) {
	for _, tc := range []struct{ op, input, method, path string }{
		{"coordination", `{"task_key":"T-2"}`, "GET", "/api/v1/tasks/T-2/coordination"},
		{"assign", `{"task_key":"T-2","request_id":"a","version":2,"mode":"role_queue","reason":"handoff"}`, "POST", "/api/v1/tasks/T-2/assign"},
		{"check_create", `{"target_key":"T-2","request_id":"c","target_version":2,"role":"reviewer"}`, "POST", "/api/v1/tasks/T-2/checks"},
	} {
		var fields map[string]json.RawMessage
		json.Unmarshal([]byte(tc.input), &fields)
		method, path, body, err := clientRoute(tc.op, fields)
		if err != nil || method != tc.method || path != tc.path {
			t.Fatal(tc.op, method, path, err)
		}
		if tc.method == "POST" {
			var forwarded map[string]json.RawMessage
			if err = json.Unmarshal(body, &forwarded); err != nil || len(forwarded["request_id"]) == 0 || len(forwarded["task_key"]) != 0 || len(forwarded["target_key"]) != 0 {
				t.Fatal("route did not retain safe mutation body")
			}
		}
	}
	for _, input := range []string{`{"task_key":"T-2","unknown":1}`, `{"task_key":"../T-2"}`} {
		var fields map[string]json.RawMessage
		json.Unmarshal([]byte(input), &fields)
		if _, _, _, err := clientRoute("coordination", fields); err == nil {
			t.Fatal("unsafe coordination route accepted")
		}
	}
}
