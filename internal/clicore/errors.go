// Package clicore holds the provider-neutral parts of the local AI CLI
// clients (internal/claudecli, internal/codexcli, internal/agycli): the
// three-lane limiter and daily call cap (Gate), stable failure codes and
// their mapping onto the llm sentinels, failure tracking for the status
// screen, and the isolated child-process runner.
package clicore

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/forgen-team/punisher/internal/llm"
)

// Provider-neutral error codes.
const (
	CodeUnauthorized  = "unauthorized"
	CodeInvalid       = "invalid_request"
	CodeQueueFull     = "queue_full"
	CodeDailyCap      = "daily_cap_reached"
	CodeClientGone    = "client_disconnected"
	maxErrMessageRune = 300
)

// Codes are one provider's stable failure codes (<provider>_unavailable, …).
type Codes struct {
	// Provider is the provider id (claude | codex | agy), used as the message prefix.
	Provider    string
	Missing     string
	NotLoggedIn string
	RateLimited string
	Timeout     string
	CLIError    string
	// LoginHint is appended to not-logged-in messages ("run `claude` once …").
	LoginHint string
}

// CodesFor builds the codes of a provider: <id>_unavailable, <id>_not_logged_in,
// <id>_rate_limited, <id>_timeout, <id>_error.
func CodesFor(provider, loginHint string) Codes {
	return Codes{Provider: provider, Missing: provider + "_unavailable", NotLoggedIn: provider + "_not_logged_in",
		RateLimited: provider + "_rate_limited", Timeout: provider + "_timeout", CLIError: provider + "_error", LoginHint: loginHint}
}

// APIError is an error with an HTTP status and a stable code.
type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// NewError builds an APIError with a truncated message.
func NewError(status int, code, msg string) *APIError {
	return &APIError{Status: status, Code: code, Message: Truncate(msg, maxErrMessageRune)}
}

var authMarkers = []string{
	"not logged in", "please run /login", "/login", "invalid api key", "oauth token",
	"authentication_error", "authentication failed", "authentication required", "unauthorized", "401", "credentials",
	"token has expired", "token expired", "invalid bearer",
}

var rateMarkers = []string{
	"usage limit", "rate limit", "rate_limit", "limit reached", "hit your limit", "429",
	"too many requests", "quota", "overloaded", "529", "resource_exhausted",
}

// Classify maps CLI failure text to an APIError of this provider.
func (c Codes) Classify(text string) *APIError {
	lower := strings.ToLower(text)
	for _, m := range authMarkers {
		if strings.Contains(lower, m) {
			return NewError(http.StatusUnauthorized, c.NotLoggedIn,
				c.Provider+" CLI is not logged in or its credentials expired ("+c.LoginHint+"): "+text)
		}
	}
	for _, m := range rateMarkers {
		if strings.Contains(lower, m) {
			return NewError(http.StatusTooManyRequests, c.RateLimited, c.Provider+" usage/rate limit: "+text)
		}
	}
	return NewError(http.StatusBadGateway, c.CLIError, text)
}

// ToLLMError maps an APIError onto the llm sentinel errors (499 = caller gone).
func (c Codes) ToLLMError(ae *APIError) error {
	if ae.Status == 499 {
		return context.Canceled
	}
	sentinel := llm.ErrServerError
	switch ae.Code {
	case c.NotLoggedIn, c.Missing:
		sentinel = llm.ErrGenerationUnavailable
	case c.RateLimited, CodeQueueFull, CodeDailyCap:
		sentinel = llm.ErrRateLimited
	case c.Timeout:
		sentinel = llm.ErrTimeout
	case CodeInvalid:
		sentinel = llm.ErrInvalidRequest
	}
	return &llm.LLMError{StatusCode: ae.Status, Code: ae.Code, Provider: c.Provider,
		Message: c.Provider + ": " + ae.Code + ": " + ae.Message, Err: sentinel}
}

// ErrorCode returns the CLI error code carried by err ("" if none).
func ErrorCode(err error) string {
	var le *llm.LLMError
	if errors.As(err, &le) {
		return le.Code
	}
	return ""
}

// Truncate trims s to maxRunes runes (with an ellipsis).
func Truncate(s string, maxRunes int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) <= maxRunes {
		return s
	}
	return string(r[:maxRunes]) + "…"
}
