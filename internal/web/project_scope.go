package web

import (
	"context"
	"strings"
)

type projectScopeKey struct{}

func WithProjectScope(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, projectScopeKey{}, id)
}
func scopedURL(ctx context.Context, path string) string {
	id, _ := ctx.Value(projectScopeKey{}).(string)
	if id == "" {
		return path
	}
	if strings.HasPrefix(path, "/web/") && !strings.HasPrefix(path, "/web/login") && !strings.HasPrefix(path, "/web/catalog/") && path != "/web/projects" {
		return "/web/p/" + id + strings.TrimPrefix(path, "/web")
	}
	if strings.HasPrefix(path, "/api/v1/") {
		return "/api/v1/p/" + id + strings.TrimPrefix(path, "/api/v1")
	}
	return path
}
func scopePage(ctx context.Context, p *browserPage) {
	for i := range p.Designs {
		p.Designs[i].DownloadURL = scopedURL(ctx, p.Designs[i].DownloadURL)
		p.Designs[i].PreviewURL = scopedURL(ctx, p.Designs[i].PreviewURL)
	}
}
func (s *Server) guarded(ctx context.Context, write bool, session string, call func() error) error {
	if s.options.AccessGuard != nil {
		return s.options.AccessGuard(ctx, write, session, call)
	}
	return call()
}
