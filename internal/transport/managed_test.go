package transport

import "testing"

func TestManagedRESTRejectsUnknownNullDuplicateQueryAndWorkerShellFields(t *testing.T) {
	f := newFixture(t)
	session := openSession(t, f, "human", "managed")
	for _, path := range []string{"poll", "report", "hosts/enroll", "start", "command"} {
		for _, body := range []string{`{"request_id":"x","command":"sh -c arbitrary"}`, `{"request_id":"x","host_secret":null}`, `{"request_id":"x","request_id":"y"}`} {
			if code, _ := request(t, f, "POST", "/api/v1/managed/"+path, session.SessionID, body); code != 400 {
				t.Fatal("invalid managed input accepted", path, code)
			}
		}
		if code, _ := request(t, f, "POST", "/api/v1/managed/"+path+"?unknown=1", session.SessionID, `{}`); code != 400 {
			t.Fatal("managed query accepted", path, code)
		}
	}
	if code, _ := request(t, f, "GET", "/api/v1/managed?x=1", session.SessionID, ""); code != 400 {
		t.Fatal("managed read query accepted")
	}
}
