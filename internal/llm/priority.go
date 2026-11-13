package llm

import (
	"context"
	"strings"
)

// Priority is the scheduling lane for a local claude CLI call:
//   - short: a brief conversational call a user is waiting on (pretest
//     question, tutor turn, Q&A, glossary). With 3+ slots one slot is kept
//     for it, so it never waits behind long generations;
//   - interactive: a user-initiated long generation (instant lecture draft
//     and review, check items). It may not take the short slot;
//   - background: scheduler/worker/pipeline calls, served after the user
//     lanes and never taking the last free slot.
type Priority string

const (
	PriorityShort       Priority = "short"
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

// ShortUnlessBackground is the lane of a short conversational call: short,
// except inside background work (ctx marked background) where it stays background.
func ShortUnlessBackground(ctx context.Context) Priority {
	if IsBackground(ctx) {
		return PriorityBackground
	}
	return PriorityShort
}

func (p Priority) normalized() Priority {
	switch Priority(strings.ToLower(strings.TrimSpace(string(p)))) {
	case PriorityBackground:
		return PriorityBackground
	case PriorityShort:
		return PriorityShort
	}
	return PriorityInteractive
}
