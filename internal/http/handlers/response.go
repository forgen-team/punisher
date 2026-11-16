package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	appMiddleware "github.com/forgen-team/punisher/internal/middleware"
	"github.com/go-chi/chi/v5/middleware"
)

const maxBodyBytes = 64 << 10

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type ErrorCode string

const (
	ErrorBadRequest   ErrorCode = "BAD_REQUEST"
	ErrorUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrorNotFound     ErrorCode = "NOT_FOUND"
	ErrorConflict     ErrorCode = "CONFLICT"
	ErrorInternal     ErrorCode = "INTERNAL_ERROR"
)

type errorEnvelope struct {
	Error errorPayload `json:"error"`
}

type errorPayload struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	RequestID string    `json:"requestId"`
}

func writeError(w http.ResponseWriter, r *http.Request, status int, code ErrorCode, message string) {
	requestID, _ := r.Context().Value(middleware.RequestIDKey).(string)
	writeJSON(w, status, errorEnvelope{
		Error: errorPayload{Code: code, Message: message, RequestID: requestID},
	})
}

// requireUser returns the authenticated user id or writes 401.
// Token auth without X-Internal-User-Id carries no user and is rejected here.
func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := appMiddleware.GetUserID(r.Context())
	if err != nil {
		writeError(w, r, http.StatusUnauthorized, ErrorUnauthorized, "user context required")
		return "", false
	}
	return userID, true
}

// decodeJSON decodes a size-limited JSON body, rejecting unknown fields.
func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("request body is empty")
		}
		return fmt.Errorf("invalid JSON body: %w", err)
	}
	return nil
}
