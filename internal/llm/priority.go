package llm

import (
	"context"
	"strings"
)

// Priority is the scheduling lane for a local claude CLI call.
// Interactive (user-initiated) calls are always served before background
// (scheduler/worker/pipeline) calls, and background calls can never take the
// last free slot.
type Priority string

const (
	PriorityInteractive Priority = "interactive"
	PriorityBackground  Priority = "background"
)

type priorityKey struct{}

// WithPriority marks ctx (and every LLM call made with a derived context) with
// a CLI lane priority. Set it once at scheduler/worker entry points so nested
// calls inherit it.
func WithPriority(ctx context.Context, p Priority) context.Context {
	return context.WithValue(ctx, priorityKey{}, p.normalized())
}

// WithBackground is shorthand for WithPriority(ctx, PriorityBackground).
func WithBackground(ctx context.Context) context.Context {
	return WithPriority(ctx, PriorityBackground)
}

// PriorityFromContext returns the priority set on ctx; interactive by default.
func PriorityFromContext(ctx context.Context) Priority {
	if ctx != nil {
		if p, ok := ctx.Value(priorityKey{}).(Priority); ok {
			return p
		}
	}
	return PriorityInteractive
}

// IsBackground reports whether ctx carries the background priority.
func IsBackground(ctx context.Context) bool {
	return PriorityFromContext(ctx) == PriorityBackground
}

func (p Priority) normalized() Priority {
	if Priority(strings.ToLower(strings.TrimSpace(string(p)))) == PriorityBackground {
		return PriorityBackground
	}
	return PriorityInteractive
}
