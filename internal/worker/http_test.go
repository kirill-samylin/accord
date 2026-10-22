package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"agent-board/internal/core"
)

func TestWorkerHTTPBindsOriginAndNeverExposesCredentialErrors(t *testing.T) {
	e := testEngine(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer synthetic-private-token" {
			t.Error("missing bearer")
		}
		if !strings.HasPrefix(r.URL.Path, "/api/v1/p/"+e.config.Project+"/") {
			t.Error("missing project scope")
		}
		http.Redirect(w, r, "http://127.0.0.1:1/secret", http.StatusTemporaryRedirect)
	}))
	defer server.Close()
	e.config.Server = server.URL
	r, err := newRemote(e.config)
	if err != nil {
		t.Fatal(err)
	}
	var output core.SessionView
	err = r.post(context.Background(), "/session", core.OpenSession{}, &output)
	if err == nil || strings.Contains(err.Error(), "synthetic-private-token") {
		t.Fatal("redirect followed or credential leaked")
	}
}

func TestEnrollmentRetryPinsSessionAndRequest(t *testing.T) {
	e := testEngine(t)
	var enrollRequests []core.ManagedHostEnrollInput
	var sessionRequests []core.OpenSession
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/session"):
			var input core.OpenSession
			json.NewDecoder(r.Body).Decode(&input)
			sessionRequests = append(sessionRequests, input)
			json.NewEncoder(w).Encode(core.SessionView{SessionID: "synthetic-human-session", ExpiresAt: time.Now().Add(time.Minute)})
		case strings.HasSuffix(r.URL.Path, "/managed/hosts/enroll"):
			var input core.ManagedHostEnrollInput
			json.NewDecoder(r.Body).Decode(&input)
			enrollRequests = append(enrollRequests, input)
			if len(enrollRequests) == 1 {
				http.Error(w, "synthetic dropped response", 500)
				return
			}
			json.NewEncoder(w).Encode(core.ManagedHost{ID: "host_test", OwnerPrincipalID: "principal_test"})
		default:
			t.Error("unexpected route")
		}
	}))
	defer server.Close()
	e.config.Server = server.URL
	if _, err := e.Enroll(context.Background()); err == nil {
		t.Fatal("expected first request ambiguity")
	}
	host, err := e.Enroll(context.Background())
	if err != nil || host.ID != "host_test" {
		t.Fatal("enrollment retry failed")
	}
	if len(enrollRequests) != 2 || enrollRequests[0].RequestID != enrollRequests[1].RequestID || enrollRequests[0].HostSecret != enrollRequests[1].HostSecret {
		t.Fatal("enrollment retries changed idempotent payload")
	}
	if len(sessionRequests) != 2 || sessionRequests[0] != sessionRequests[1] {
		t.Fatal("enrollment retry changed session idempotency scope")
	}
}
