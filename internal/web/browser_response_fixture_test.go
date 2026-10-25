package web

import (
	"agent-board/internal/core"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"reflect"
	"sync"
)

type captureBrowserResponse struct {
	mu   sync.Mutex
	page Page
	name string
}

// Fixtures inspect the actual public JSON projection, never a renderer hook.
var browserResponses sync.Map

func observeBrowserResponse(s *Server, w http.ResponseWriter, r *http.Request) {
	r.Header.Set("Accept", "application/json")
	s.ServeHTTP(w, r)
	value, ok := browserResponses.Load(filepath.Clean(s.options.DataDir))
	if !ok {
		return
	}
	var envelope browserEnvelope
	result := w.(interface{ Result() *http.Response }).Result()
	body, _ := io.ReadAll(result.Body)
	if json.Unmarshal(body, &envelope) != nil || envelope.Page.Name == "" {
		return
	}
	c := value.(*captureBrowserResponse)
	c.mu.Lock()
	defer c.mu.Unlock()
	c.page = pageFromBrowserJSON(envelope.Page)
	c.page.Error = envelope.Error.Message
	c.name = envelope.Page.Name
}
func pageFromBrowserJSON(in browserPage) Page {
	var out Page
	source, target := reflect.ValueOf(in), reflect.ValueOf(&out).Elem()
	for n := 0; n < target.NumField(); n++ {
		field := source.FieldByName(target.Type().Field(n).Name)
		if field.IsValid() && field.Type().AssignableTo(target.Field(n).Type()) {
			target.Field(n).Set(field)
		}
	}
	out.Principal = core.Principal{Owner: in.Principal.Owner, Machine: in.Principal.Machine}
	return out
}
func (c *captureBrowserResponse) last() Page { c.mu.Lock(); defer c.mu.Unlock(); return c.page }
