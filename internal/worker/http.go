package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"sort"
	"time"

	"agent-board/internal/client"
	"agent-board/internal/core"
)

type remote struct {
	client  *http.Client
	origin  *url.URL
	project string
	session core.SessionView
}

type statusError struct{ code int }

func (e statusError) Error() string {
	return "Accord rejected the worker request; check local enrollment, project access and server capability"
}

func newRemote(config Config) (*remote, error) {
	origin, err := client.Origin(config.Server)
	if err != nil {
		return nil, err
	}
	if config.Project == "" {
		return nil, errors.New("managed worker requires an explicit project ID")
	}
	if err = client.Project(config.Project); err != nil {
		return nil, err
	}
	token, err := ReadToken(config.TokenFile)
	if err != nil {
		return nil, err
	}
	return &remote{origin: origin, project: config.Project, client: client.HTTP(origin, token, 3*time.Second)}, nil
}

func (r *remote) post(ctx context.Context, path string, input, output any) error {
	return r.request(ctx, http.MethodPost, path, input, output)
}

func (r *remote) request(ctx context.Context, method, path string, input, output any) error {
	data, err := json.Marshal(input)
	if err != nil {
		return errors.New("cannot encode managed request")
	}
	if method == http.MethodGet {
		data = nil
	}
	target := *r.origin
	target.Path = "/api/v1/p/" + r.project + path
	request, err := http.NewRequestWithContext(ctx, method, target.String(), bytes.NewReader(data))
	if err != nil {
		return errors.New("cannot prepare managed request")
	}
	if method != http.MethodGet {
		request.Header.Set("Content-Type", "application/json")
	}
	if r.session.SessionID != "" {
		request.Header.Set("X-Session", r.session.SessionID)
	}
	response, err := r.client.Do(request)
	if err != nil {
		return errors.New("Accord connection unavailable; local ownership may be uncertain")
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return statusError{code: response.StatusCode}
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, client.MaxResponseBytes+1))
	if err = decoder.Decode(output); err != nil {
		return errors.New("invalid Accord managed response")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return errors.New("oversized or trailing Accord response")
	}
	return nil
}

func (r *remote) open(ctx context.Context, config Config, workerID string) error {
	input := core.OpenSession{RequestID: randomID("session_"), Role: "human", RunID: workerID, Machine: config.Machine}
	return r.post(ctx, "/session", input, &r.session)
}

func (r *remote) enrollSession(ctx context.Context, config Config, id string) error {
	input := core.OpenSession{RequestID: id + "_session", Role: "human", RunID: id, Machine: config.Machine}
	return r.post(ctx, "/session", input, &r.session)
}

func (r *remote) heartbeat(ctx context.Context) error {
	input := core.HeartbeatInput{RequestID: randomID("heartbeat_"), RoleGeneration: r.session.RoleGeneration}
	var response core.SessionView
	return r.post(ctx, "/heartbeat", input, &response)
}

func (e *Engine) Enroll(ctx context.Context) (core.ManagedHost, error) {
	if err := e.probe(e.config); err != nil {
		return core.ManagedHost{}, err
	}
	r, err := newRemote(e.config)
	if err != nil {
		return core.ManagedHost{}, err
	}
	if err = r.enrollSession(ctx, e.config, e.journal.EnrollRequestID); err != nil {
		return core.ManagedHost{}, err
	}
	e.mu.Lock()
	if e.journal.HostID != "" {
		e.mu.Unlock()
		return core.ManagedHost{}, errors.New("this state directory is already enrolled; use worker run")
	}
	if e.journal.Enrollment == nil {
		input := core.ManagedHostEnrollInput{RequestID: e.journal.EnrollRequestID, Name: e.config.Name, HostSecret: e.journal.HostSecret, Controllers: e.config.Controllers}
		for alias := range e.config.Workspaces {
			input.Workspaces = append(input.Workspaces, alias)
		}
		for provider := range e.config.Executables {
			input.Providers = append(input.Providers, provider)
		}
		sort.Strings(input.Workspaces)
		sort.Strings(input.Providers)
		e.journal.Enrollment = &input
		if err = e.save(); err != nil {
			e.mu.Unlock()
			return core.ManagedHost{}, err
		}
	}
	input := *e.journal.Enrollment
	e.mu.Unlock()
	var host core.ManagedHost
	if err = r.post(ctx, "/managed/hosts/enroll", input, &host); err != nil {
		return host, err
	}
	if host.ID == "" || host.OwnerPrincipalID == "" {
		return host, errors.New("invalid host enrollment response")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	e.journal.HostID = host.ID
	e.journal.OwnerPrincipalID = host.OwnerPrincipalID
	return host, e.save()
}
