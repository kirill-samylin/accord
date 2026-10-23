package web

import (
	"agent-board/internal/core"
	"net/http"
	"net/url"
	"time"
)

// A scope read uses current browser/project authority. The summary list is never
// expanded into guessed role/check permissions and no browser action is issued.
func (s *Server) automationHostScopePage(r *http.Request, b *browserSession) (Page, error) {
	p := Page{Title: "Разрешение папки", View: "automation-host-scope", Principal: b.Principal, Session: b.Session, CSRF: b.CSRF, CanContinue: !b.Session.ExpiresAt.After(time.Now()), Actions: map[string]string{}, Draft: map[string]string{}}
	if p.CanContinue {
		return p, nil
	}
	q, e := url.ParseQuery(r.URL.RawQuery)
	if e != nil {
		return p, core.Fail("validation", "Неверные параметры папки")
	}
	for key, values := range q {
		if len(values) != 1 || key != "host_id" && key != "workspace" {
			return p, core.Fail("validation", "Неизвестный или повторный параметр папки")
		}
	}
	if q.Get("host_id") == "" || q.Get("workspace") == "" || len(q.Get("host_id")) > 128 || len(q.Get("workspace")) > 128 {
		return p, core.Fail("validation", "Выберите компьютер и папку")
	}
	m, ok := s.board.(managedBoard)
	if !ok {
		return p, core.Fail("not_found", "Подключённые компьютеры недоступны")
	}
	data, e := m.ManagedList(r.Context(), b.actor(), core.ManagedListInput{HostID: q.Get("host_id"), Workspace: q.Get("workspace")})
	if e != nil {
		return p, e
	}
	if len(data.Hosts) != 1 || data.Hosts[0].ID != q.Get("host_id") {
		return p, core.Fail("not_found", "Компьютер недоступен")
	}
	if s.options.ProjectInfo != nil {
		project, e := s.options.ProjectInfo(r.Context())
		if e != nil {
			return p, e
		}
		p.Project = project
		data.Hosts[0].CanControl = data.Hosts[0].CanControl && project.CanWrite
	}
	p.Managed = &data
	return p, nil
}
