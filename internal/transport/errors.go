package transport

import (
	"errors"
	"net/http"
	"strings"

	"agent-board/internal/core"
)

type errorResponse struct {
	Error core.Error `json:"error"`
}

func publicError(err error) (errorResponse, int) {
	var domain *core.Error
	if !errors.As(err, &domain) {
		return errorResponse{core.Error{Code: "INTERNAL", Message: "internal server error"}}, http.StatusInternalServerError
	}
	code := strings.ToUpper(domain.Code)
	status := http.StatusConflict
	switch code {
	case "VALIDATION", "INVALID_INPUT", "INVALID_TRANSITION", "EVIDENCE_MISSING", "CYCLE":
		status = http.StatusBadRequest
	case "INTERNAL":
		status = http.StatusInternalServerError
	case "UNAUTHORIZED":
		status = http.StatusUnauthorized
	case "FORBIDDEN", "FORBIDDEN_ROLE":
		status = http.StatusForbidden
	case "NOT_FOUND", "NO_SESSION":
		status = http.StatusNotFound
	}
	return errorResponse{core.Error{Code: code, Message: domain.Message}}, status
}
