package projects

import (
	"agent-board/internal/core"
	"agent-board/internal/store"
	"agent-board/internal/transport"
	"agent-board/internal/web"
	"context"
	"net/url"
	"os"
	"path/filepath"
)

func (r *Registry) configure(v *runtime, id string) {
	v.service = &service{registry: r, projectID: id, board: v.store}
	namespace := id
	if id == r.defaultID {
		namespace = ""
	}
	v.browser.ConfigureProject(v.service, web.Options{ProjectID: namespace, ProjectInfo: v.service.CurrentProject, AccessGuard: v.service.Guard, ActionEpoch: v.service.Epoch, SessionAllowed: v.service.SessionAllowed})
	origin, _ := url.Parse(r.options.Origin)
	external := origin.Host
	if transport.ValidateExternalHost(external) != nil {
		external = ""
	}
	v.native = transport.NewHandlerWithOptions(v.service, nil, transport.HandlerOptions{Extra: v.browser, Designs: v.browser, ExternalHost: external})
}
func (r *Registry) installDefault() error {
	v := &runtime{store: r.options.Default, browser: r.options.Browser}
	r.configure(v, r.defaultID)
	r.runtimes[r.defaultID] = v
	return nil
}
func (r *Registry) runtime(id string) (*runtime, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v := r.runtimes[id]; v != nil {
		return v, nil
	}
	db, dir, err := r.projectPaths(id)
	if err != nil {
		return nil, err
	}
	// IDs originate in the registry, never in a browser supplied filesystem path.
	if _, err = os.Stat(db); err != nil {
		return nil, notFound()
	}
	return r.openRuntime(id, db, dir, true)
}
func (r *Registry) openRuntime(id, db, dir string, published bool) (*runtime, error) {
	if published {
		if err := validateProjectStorage(db, dir); err != nil {
			return nil, err
		}
	}
	cfg := r.options.Config
	cfg.Project = id
	cfg.CursorNamespace = id
	board, err := store.Open(db, cfg, nil)
	if err != nil {
		return nil, err
	}
	if r.options.Offline {
		v := &runtime{store: board}
		r.runtimes[id] = v
		return v, nil
	}
	browser, err := web.New(board, web.Options{Origin: r.options.Origin, DataDir: dir, TrustedProxy: r.options.TrustedProxy})
	if err != nil {
		board.Close()
		return nil, err
	}
	v := &runtime{store: board, browser: browser}
	r.configure(v, id)
	r.runtimes[id] = v
	return v, nil
}
func (r *Registry) provision(id string) (*runtime, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if v := r.runtimes[id]; v != nil {
		return v, nil
	}
	db, dir, err := r.projectPaths(id)
	if err != nil {
		return nil, err
	}
	if _, err = privateRoot(filepath.Dir(db)); err != nil {
		return nil, err
	}
	return r.openRuntime(id, db, dir, false)
}
func (r *Registry) localPrincipal(ctx context.Context, v *runtime, who Identity) (core.Principal, error) {
	var authority *core.Principal
	if v.store == r.options.Default && who.Source == "legacy" {
		authority = &core.Principal{ID: who.TokenID, Owner: who.Owner, Machine: who.Machine}
	}
	return v.store.ProjectPrincipal(ctx, who.Owner, who.Machine, who.Expires, authority)
}

// Make every newly created ancestor durable before publishing ACTIVE in the
// already durable registry. Database FULL sync alone cannot persist ancestors.
func (r *Registry) syncProvision(id string) error {
	base := filepath.Join(r.options.DataDir, "projects", id)
	for _, path := range []string{filepath.Join(base, "web", "blobs"), filepath.Join(base, "web", "staging"), filepath.Join(base, "web"), base, filepath.Dir(base), r.options.DataDir} {
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		err = f.Sync()
		f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}
