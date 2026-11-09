// Package learning holds pure, DB-free rules for learning progress:
// concept state transitions, frontier computation and study-request keys.
package learning

import (
	"errors"
	"fmt"
)

// Concept learning states (design §3.4). A missing row means NotStarted.
const (
	NotStarted = "not_started"
	InProgress = "in_progress"
	Completed  = "completed"
	Known      = "known"
)

// ErrInvalidState is returned for an unknown requested state.
var ErrInvalidState = errors.New("invalid concept state")

// IsValidState reports whether s is one of the four concept states.
func IsValidState(s string) bool {
	switch s {
	case NotStarted, InProgress, Completed, Known:
		return true
	}
	return false
}

// IsDone reports whether a state counts as learned (completed or known).
// Done concepts are excluded from recommendations and generation.
func IsDone(state string) bool {
	return state == Completed || state == Known
}

// NormalizeState maps the empty string (no row) to NotStarted.
func NormalizeState(state string) string {
	if state == "" {
		return NotStarted
	}
	return state
}

// NextState applies an explicit user request to the current state.
//
// Rules:
//   - completed / known are only ever set by this explicit call;
//   - in_progress never downgrades a done concept (opening a finished
//     lecture again keeps it completed) — the request is a no-op;
//   - not_started is an explicit reset (undo) and is always allowed;
//   - requesting the current state is a no-op.
func NextState(current, requested string) (next string, changed bool, err error) {
	current = NormalizeState(current)
	if !IsValidState(requested) {
		return current, false, fmt.Errorf("%w: %q", ErrInvalidState, requested)
	}
	if requested == current {
		return current, false, nil
	}
	if requested == InProgress && IsDone(current) {
		return current, false, nil
	}
	return requested, true, nil
}
