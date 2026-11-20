package clicore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/llm"
)

// GateConfig sizes a Gate. Zero values get sane defaults.
type GateConfig struct {
	// Concurrency is the total number of parallel CLI processes (1..4,
	// default 3). From 3 one slot is kept for short conversational calls
	// (pretest, tutor, Q&A, glossary): long generations never take it.
	Concurrency int
	// BackgroundConcurrency caps parallel background calls (default 1). With
	// Concurrency >= 2 background never takes the last free slot.
	BackgroundConcurrency int
	InteractiveQueue      int           // interactive waiters before failing (default 4)
	ShortQueue            int           // short-lane waiters before failing (default 4)
	BackgroundQueue       int           // background waiters before failing (default 8)
	InteractiveQueueWait  time.Duration // default 45s
	BackgroundQueueWait   time.Duration // default 5m
	DefaultTimeout        time.Duration // default 180s
	MaxTimeout            time.Duration // default 600s
	// DailyLimit caps CLI calls per local day; 0 = unlimited (default).
	DailyLimit int
}

func (c GateConfig) normalized() GateConfig {
	if c.Concurrency < 1 {
		c.Concurrency = 3
	}
	if c.Concurrency > 4 {
		c.Concurrency = 4
	}
	if c.BackgroundConcurrency < 1 {
		c.BackgroundConcurrency = 1
	}
	if c.BackgroundConcurrency > c.Concurrency {
		c.BackgroundConcurrency = c.Concurrency
	}
	if c.InteractiveQueue <= 0 {
		c.InteractiveQueue = 4
	}
	if c.ShortQueue <= 0 {
		c.ShortQueue = 4
	}
	if c.BackgroundQueue <= 0 {
		c.BackgroundQueue = 8
	}
	if c.InteractiveQueueWait <= 0 {
		c.InteractiveQueueWait = 45 * time.Second
	}
	if c.BackgroundQueueWait <= 0 {
		c.BackgroundQueueWait = 5 * time.Minute
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 180 * time.Second
	}
	if c.MaxTimeout < c.DefaultTimeout {
		c.MaxTimeout = 600 * time.Second
	}
	return c
}

// Gate admits CLI calls: lane limiter, call timeout and the optional daily
// call cap. One Gate may be shared by several provider clients so the
// machine-wide process limit holds whichever AI is selected.
type Gate struct {
	cfg     GateConfig
	limiter *Limiter

	mu       sync.Mutex
	dayKey   string
	dayCalls int
}

// NewGate builds a gate.
func NewGate(cfg GateConfig) *Gate {
	cfg = cfg.normalized()
	return &Gate{cfg: cfg, limiter: NewLimiter(LimiterConfig{
		Total: cfg.Concurrency, BackgroundMax: cfg.BackgroundConcurrency,
		InteractiveQueue: cfg.InteractiveQueue, BackgroundQueue: cfg.BackgroundQueue, ShortQueue: cfg.ShortQueue,
	})}
}

// Limiter is the gate's lane limiter (stats for status/health).
func (g *Gate) Limiter() *Limiter { return g.limiter }

// Config is the normalized configuration.
func (g *Gate) Config() GateConfig { return g.cfg }

// Timeout is the call timeout of req: its own (default DefaultTimeout), capped at MaxTimeout.
func (g *Gate) Timeout(req llm.MessagesRequest) time.Duration {
	timeout := g.cfg.DefaultTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	if timeout > g.cfg.MaxTimeout {
		timeout = g.cfg.MaxTimeout
	}
	return timeout
}

// LaneOf is the lane of req: its Priority, else the one set on ctx.
func LaneOf(ctx context.Context, req llm.MessagesRequest) Lane {
	priority := req.Priority
	if priority == "" {
		priority = llm.PriorityFromContext(ctx)
	}
	return ParseLane(string(priority))
}

// Admit waits for a lane slot for req and applies the daily cap. It returns
// the run context (the call timeout) and a done func releasing the slot.
//   - short / interactive: queue wait counts against the call timeout and is capped by
//     InteractiveQueueWait, so a user call fails fast behind other work;
//   - background: queue wait (BackgroundQueueWait) is separate and the call
//     timeout starts once a slot is granted.
func (g *Gate) Admit(parent context.Context, req llm.MessagesRequest, codes Codes) (context.Context, func(), *APIError) {
	lane := LaneOf(parent, req)
	timeout := g.Timeout(req)
	wait := g.cfg.InteractiveQueueWait
	var runCtx context.Context
	var runCancel context.CancelFunc
	queueParent := parent
	if lane == LaneBackground {
		wait = g.cfg.BackgroundQueueWait
	} else {
		runCtx, runCancel = context.WithTimeout(parent, timeout)
		queueParent = runCtx
	}
	fail := func(e *APIError) (context.Context, func(), *APIError) {
		if runCancel != nil {
			runCancel()
		}
		return nil, nil, e
	}
	qctx, qcancel := context.WithTimeout(queueParent, wait)
	release, err := g.limiter.Acquire(qctx, lane)
	qcancel()
	if errors.Is(err, ErrQueueFull) {
		return fail(NewError(http.StatusTooManyRequests, CodeQueueFull, fmt.Sprintf("%s queue is full", lane)))
	}
	if err != nil {
		if parent.Err() != nil {
			return fail(NewError(499, CodeClientGone, "caller gave up while queued"))
		}
		return fail(NewError(http.StatusGatewayTimeout, codes.Timeout, fmt.Sprintf("timed out waiting in %s queue", lane)))
	}
	if !g.reserveDaily() {
		release()
		return fail(NewError(http.StatusTooManyRequests, CodeDailyCap, fmt.Sprintf("daily call cap (%d) reached", g.cfg.DailyLimit)))
	}
	if runCtx == nil {
		runCtx, runCancel = context.WithTimeout(parent, timeout)
	}
	return runCtx, func() { runCancel(); release() }, nil
}

func (g *Gate) reserveDaily() bool {
	if g.cfg.DailyLimit <= 0 {
		return true
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	key := time.Now().Format("2006-01-02")
	if g.dayKey != key {
		g.dayKey, g.dayCalls = key, 0
	}
	if g.dayCalls >= g.cfg.DailyLimit {
		return false
	}
	g.dayCalls++
	return true
}
