package worker

import (
	"context"
	"errors"
	"net/http"
	"sort"

	"agent-board/internal/core"
)

func (e *Engine) Identity(ctx context.Context) (string, error) {
	r, err := newRemote(e.config)
	if err != nil {
		return "", err
	}
	if err = r.open(ctx, e.config, e.journal.WorkerID); err != nil {
		return "", err
	}
	var page core.ManagedPage
	if err = r.request(ctx, http.MethodGet, "/managed", nil, &page); err != nil {
		return "", err
	}
	if page.ActorPrincipalID == "" {
		return "", errors.New("server does not expose project-local actor identity")
	}
	return page.ActorPrincipalID, nil
}

// Consent is an explicit local operation. A browser cannot grant itself host
// control or change the locally selected executable and workspace mapping.
func (e *Engine) Consent(ctx context.Context) (core.ManagedHost, error) {
	r, err := newRemote(e.config)
	if err != nil {
		return core.ManagedHost{}, err
	}
	if e.journal.HostID == "" {
		return core.ManagedHost{}, errors.New("enroll the host before updating local consent")
	}
	if e.journal.Consent != nil {
		return e.sendConsent(ctx, r)
	}
	if err = r.open(ctx, e.config, e.journal.WorkerID); err != nil {
		return core.ManagedHost{}, err
	}
	var page core.ManagedPage
	if err = r.request(ctx, http.MethodGet, "/managed", nil, &page); err != nil {
		return core.ManagedHost{}, err
	}
	var current *core.ManagedHost
	for i := range page.Hosts {
		if page.Hosts[i].ID == e.journal.HostID {
			current = &page.Hosts[i]
		}
	}
	if current == nil {
		return core.ManagedHost{}, errors.New("local host is absent from the project")
	}
	input := core.ManagedHostUpdateInput{HostID: current.ID, Version: current.Version, State: "active", ManagedHostEnrollInput: core.ManagedHostEnrollInput{RequestID: randomID("consent_"), Name: e.config.Name, HostSecret: e.journal.HostSecret, Controllers: e.config.Controllers}}
	for alias := range e.config.Workspaces {
		input.Workspaces = append(input.Workspaces, alias)
	}
	for provider := range e.config.Executables {
		input.Providers = append(input.Providers, provider)
	}
	sort.Strings(input.Workspaces)
	sort.Strings(input.Providers)
	e.mu.Lock()
	e.journal.Consent = &input
	err = e.save()
	e.mu.Unlock()
	if err != nil {
		return core.ManagedHost{}, err
	}
	return e.sendConsent(ctx, r)
}

func (e *Engine) sendConsent(ctx context.Context, r *remote) (core.ManagedHost, error) {
	input := *e.journal.Consent
	if err := r.enrollSession(ctx, e.config, input.RequestID); err != nil {
		return core.ManagedHost{}, err
	}
	var host core.ManagedHost
	if err := r.post(ctx, "/managed/hosts/update", input, &host); err != nil {
		return host, err
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal.Consent = nil
	pins, err := pinPaths(e.config)
	if err != nil {
		return host, err
	}
	e.journal.PathPins = pins
	return host, e.save()
}
