package transport

import (
	"context"
	"net/http"
	"net/url"
	"strconv"

	"agent-board/internal/core"
)

type configReader interface {
	ReadConfig(context.Context, core.Principal) (core.ConfigView, error)
	ReadConfigHistory(context.Context, core.Principal, int64, int) (core.ConfigHistory, error)
}

func registerConfig(mux *http.ServeMux, s Service) {
	reader, ok := s.(configReader)
	if !ok {
		return
	}
	mux.HandleFunc("GET /api/v1/config", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "" || r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
			writeError(w, core.Fail("VALIDATION", "config accepts no query or body"))
			return
		}
		p, ok := r.Context().Value(principalKey{}).(core.Principal)
		if !ok {
			unauthorized(w)
			return
		}
		v, err := reader.ReadConfig(r.Context(), p)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
	})
	mux.HandleFunc("GET /api/v1/config/history", func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength > 0 || len(r.TransferEncoding) > 0 {
			writeError(w, core.Fail("VALIDATION", "history accepts no body"))
			return
		}
		q, parseErr := url.ParseQuery(r.URL.RawQuery)
		if parseErr != nil {
			writeError(w, core.Fail("VALIDATION", "malformed query"))
			return
		}
		limit := 100
		after := int64(0)
		for key, values := range q {
			if len(values) != 1 || (key != "limit" && key != "after_version") {
				writeError(w, core.Fail("VALIDATION", "unknown or repeated query parameter"))
				return
			}
		}
		var err error
		if raw := q.Get("limit"); raw != "" {
			limit, err = strconv.Atoi(raw)
			if err != nil {
				writeError(w, core.Fail("VALIDATION", "invalid limit"))
				return
			}
		}
		if raw := q.Get("after_version"); raw != "" {
			after, err = strconv.ParseInt(raw, 10, 64)
			if err != nil {
				writeError(w, core.Fail("VALIDATION", "invalid version"))
				return
			}
		}
		p, ok := r.Context().Value(principalKey{}).(core.Principal)
		if !ok {
			unauthorized(w)
			return
		}
		v, err := reader.ReadConfigHistory(r.Context(), p, after, limit)
		if err != nil {
			writeError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, v)
	})
}
