// Package llm holds the generation request/result types and error sentinels
// shared by services and the local Claude Code CLI client (internal/claudecli).
package llm

import (
	"errors"
	"time"
)

// Standard errors.
var (
	ErrAPIKeyRequired = errors.New("API_KEY_REQUIRED")
	ErrRateLimited    = errors.New("RATE_LIMITED")
	ErrInvalidRequest = errors.New("INVALID_REQUEST")
	ErrTimeout        = errors.New("TIMEOUT")
	ErrServerError    = errors.New("SERVER_ERROR")
	// ErrGenerationUnavailable: the local claude CLI is missing or not logged in.
	// Callers record a failure; there is no fallback provider.
	ErrGenerationUnavailable = errors.New("GENERATION_UNAVAILABLE")
)

// LLMError is an error with a status and a stable code.
type LLMError struct {
	StatusCode int
	Message    string
	Err        error
	// Code is the CLI failure code (claude_not_logged_in, claude_unavailable, ...).
	Code string
}

func (e *LLMError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *LLMError) Unwrap() error { return e.Err }

// Message is one conversation turn.
type Message struct {
	Role    string `json:"role"` // "user" | "assistant"
	Content string `json:"content"`
}

// MessagesRequest is one generation call. Either User or Messages must be set;
// when both are set User is appended as the final user turn.
type MessagesRequest struct {
	Model       string
	System      string
	User        string
	Messages    []Message
	MaxTokens   int
	Temperature float64
	Timeout     time.Duration // whole-call timeout; default client timeout
	// Priority is the scheduling lane. Empty = taken from ctx (WithPriority), else interactive.
	Priority Priority
}

// Usage is the token accounting reported by the CLI.
type Usage struct {
	InputTokens  int `json:"inputTokens"`
	OutputTokens int `json:"outputTokens"`
}

// Total returns input + output tokens.
func (u Usage) Total() int { return u.InputTokens + u.OutputTokens }

// MessagesResult is the outcome of a generation call.
type MessagesResult struct {
	Text       string
	Model      string
	StopReason string // end_turn | max_tokens | ...
	Usage      Usage
	LatencyMs  int64
}

// Truncated reports whether generation stopped because max_tokens was reached.
func (r MessagesResult) Truncated() bool { return r.StopReason == "max_tokens" }
