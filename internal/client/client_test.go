package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func TestOriginPolicyAndCredentialRedirectIsolation(t *testing.T) {
	for _, raw := range []string{"http://example.com:80", "http://localhost:80", "ftp://127.0.0.1:80", "https://user:secret@example.com", "https://example.com/path", "https://example.com?x=1", "https://example.com?", "https://example.com#x"} {
		if _, err := Origin(raw); err == nil {
			t.Fatalf("accepted unsafe origin %q", raw)
		}
	}
	for _, raw := range []string{"https://example.com", "http://127.0.0.1:8080", "http://[::1]:8080"} {
		if _, err := Origin(raw); err != nil {
			t.Fatalf("rejected origin %q", raw)
		}
	}
	var destinationCalls atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { destinationCalls.Add(1) }))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, 302) }))
	defer source.Close()
	origin, err := Origin(source.URL)
	if err != nil {
		t.Fatal(err)
	}
	c := HTTP(origin, "synthetic-secret", time.Second)
	response, err := c.Get(source.URL)
	if response != nil {
		response.Body.Close()
	}
	if err == nil || destinationCalls.Load() != 0 {
		t.Fatal("followed credentialed redirect")
	}
	request, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, destination.URL, nil)
	response, err = c.Do(request)
	if response != nil {
		response.Body.Close()
	}
	if err == nil || destinationCalls.Load() != 0 {
		t.Fatal("transport sent credential off-origin")
	}
}

func TestTokenPrivacyAndBounds(t *testing.T) {
	t.Setenv("AGENT_BOARD_TOKEN", "synthetic-env")
	if token, err := ReadToken(""); err != nil || token != "synthetic-env" {
		t.Fatal("env token unavailable")
	}
	dir := t.TempDir()
	private := filepath.Join(dir, "private.token")
	if err := os.WriteFile(private, []byte("synthetic-file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if token, err := ReadToken(private); err != nil || token != "synthetic-file" {
		t.Fatal("private file unavailable")
	}
	link := filepath.Join(dir, "link.token")
	if err := os.Symlink(private, link); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadToken(link); err == nil {
		t.Fatal("symlink token accepted")
	}
	if err := os.Chmod(private, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadToken(private); err == nil {
		t.Fatal("public token accepted")
	}
	if err := os.Chmod(private, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(private, make([]byte, 4097), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadToken(private); err == nil {
		t.Fatal("oversized token accepted")
	}
}
