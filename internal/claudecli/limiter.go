package claudecli

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// ErrQueueFull is returned when a lane's wait queue is full.
var ErrQueueFull = errors.New("queue full")

// Lane is a scheduling class. Short = brief conversational user calls
// (pretest question, tutor, Q&A, glossary); interactive = user-initiated long
// generations; background = scheduler/worker/pipeline calls.
type Lane string

const (
	LaneShort       Lane = "short"
	LaneInteractive Lane = "interactive"
	LaneBackground  Lane = "background"
)

// lanes in service order.
var lanes = []Lane{LaneShort, LaneInteractive, LaneBackground}

// ParseLane maps a request priority to a lane; anything but "background" or
// "short" is interactive so an unknown/empty value can never demote a user call.
func ParseLane(raw string) Lane {
	switch {
	case strings.EqualFold(strings.TrimSpace(raw), string(LaneBackground)):
		return LaneBackground
	case strings.EqualFold(strings.TrimSpace(raw), string(LaneShort)):
		return LaneShort
	}
	return LaneInteractive
}

// LimiterConfig sizes the lanes.
type LimiterConfig struct {
	Total            int // total parallel CLI processes (all lanes)
	BackgroundMax    int // max parallel background processes
	InteractiveQueue int // interactive waiters before 429
	BackgroundQueue  int // background waiters before 429
	ShortQueue       int // short waiters before 429 (0 = InteractiveQueue)
}

// Limiter is a three-lane semaphore:
//   - waiters are served FIFO per lane, short first, then interactive, then
//     background;
//   - with Total >= 3 one slot is kept for the short lane: interactive and
//     background together hold at most Total-1, so a short call never waits
//     behind long generations (with Total <= 2 short shares with interactive);
//   - background holds at most BackgroundMax slots and, when Total >= 2,
//     never takes the last free slot, so a user call always finds one;
//   - each lane has its own bounded queue.
type Limiter struct {
	cfg LimiterConfig

	mu      sync.Mutex
	active  map[Lane]int
	waiters map[Lane][]*waiter
}

type waiter struct {
	ready   chan struct{}
	granted bool
}

// NewLimiter builds a limiter; values are clamped to sane bounds.
func NewLimiter(cfg LimiterConfig) *Limiter {
	if cfg.Total < 1 {
		cfg.Total = 1
	}
	if cfg.BackgroundMax < 0 {
		cfg.BackgroundMax = 0
	}
	if cfg.BackgroundMax > cfg.Total {
		cfg.BackgroundMax = cfg.Total
	}
	if cfg.InteractiveQueue < 0 {
		cfg.InteractiveQueue = 0
	}
	if cfg.BackgroundQueue < 0 {
		cfg.BackgroundQueue = 0
	}
	if cfg.ShortQueue <= 0 {
		cfg.ShortQueue = cfg.InteractiveQueue
	}
	return &Limiter{cfg: cfg, active: map[Lane]int{}, waiters: map[Lane][]*waiter{}}
}

// reserved is the number of slots background may never use.
func (l *Limiter) reserved() int {
	if l.cfg.Total >= 2 {
		return 1
	}
	return 0
}

// shortReserved is the number of slots only the short lane may use.
func (l *Limiter) shortReserved() int {
	if l.cfg.Total >= 3 {
		return 1
	}
	return 0
}

// longActive counts the slots held by the long lanes (interactive + background).
func (l *Limiter) longActive() int { return l.active[LaneInteractive] + l.active[LaneBackground] }

func (l *Limiter) totalActive() int { return l.active[LaneShort] + l.longActive() }

// canGrant reports whether the head waiter of lane may start now. Caller holds mu.
func (l *Limiter) canGrant(lane Lane) bool {
	if l.totalActive() >= l.cfg.Total {
		return false
	}
	if lane == LaneShort {
		return true
	}
	if l.longActive() >= l.cfg.Total-l.shortReserved() {
		return false
	}
	if lane == LaneInteractive {
		return len(l.waiters[LaneShort]) == 0
	}
	return len(l.waiters[LaneShort]) == 0 && len(l.waiters[LaneInteractive]) == 0 &&
		l.active[LaneBackground] < l.cfg.BackgroundMax &&
		l.totalActive() < l.cfg.Total-l.reserved()
}

// dispatch grants slots to queue heads, short first. Caller holds mu.
func (l *Limiter) dispatch() {
	for _, lane := range lanes {
		for len(l.waiters[lane]) > 0 && l.canGrant(lane) {
			w := l.waiters[lane][0]
			l.waiters[lane] = l.waiters[lane][1:]
			w.granted = true
			l.active[lane]++
			close(w.ready)
		}
	}
}

func (l *Limiter) queueCap(lane Lane) int {
	switch lane {
	case LaneBackground:
		return l.cfg.BackgroundQueue
	case LaneShort:
		return l.cfg.ShortQueue
	}
	return l.cfg.InteractiveQueue
}

// Acquire waits for a slot in lane. It fails fast with ErrQueueFull when the
// lane queue is full and returns ctx.Err() if the caller gives up waiting.
func (l *Limiter) Acquire(ctx context.Context, lane Lane) (release func(), err error) {
	l.mu.Lock()
	if len(l.waiters[lane]) == 0 && l.canGrant(lane) {
		l.active[lane]++
		l.mu.Unlock()
		return l.releaser(lane), nil
	}
	if len(l.waiters[lane]) >= l.queueCap(lane) {
		l.mu.Unlock()
		return nil, ErrQueueFull
	}
	w := &waiter{ready: make(chan struct{})}
	l.waiters[lane] = append(l.waiters[lane], w)
	l.mu.Unlock()

	select {
	case <-w.ready:
		return l.releaser(lane), nil
	case <-ctx.Done():
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if w.granted { // granted concurrently with cancellation: give the slot back
		l.active[lane]--
		l.dispatch()
		return nil, ctx.Err()
	}
	q := l.waiters[lane]
	for i, x := range q {
		if x == w {
			l.waiters[lane] = append(q[:i:i], q[i+1:]...)
			break
		}
	}
	// A cancelled user-lane waiter may have been blocking another lane.
	l.dispatch()
	return nil, ctx.Err()
}

func (l *Limiter) releaser(lane Lane) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			l.mu.Lock()
			l.active[lane]--
			l.dispatch()
			l.mu.Unlock()
		})
	}
}

// LaneStats is the per-lane snapshot exposed in /health.
type LaneStats struct {
	Active    int `json:"active"`
	Queued    int `json:"queued"`
	MaxActive int `json:"maxActive"`
	QueueSize int `json:"queueSize"`
}

// Stats returns total active and queued counts across lanes.
func (l *Limiter) Stats() (active, queued int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.totalActive(), len(l.waiters[LaneShort]) + len(l.waiters[LaneInteractive]) + len(l.waiters[LaneBackground])
}

// LaneStats returns per-lane counts.
func (l *Limiter) LaneStats() map[Lane]LaneStats {
	l.mu.Lock()
	defer l.mu.Unlock()
	longMax := l.cfg.Total - l.shortReserved()
	bgMax := min(l.cfg.BackgroundMax, l.cfg.Total-l.reserved(), longMax)
	return map[Lane]LaneStats{
		LaneShort:       {Active: l.active[LaneShort], Queued: len(l.waiters[LaneShort]), MaxActive: l.cfg.Total, QueueSize: l.cfg.ShortQueue},
		LaneInteractive: {Active: l.active[LaneInteractive], Queued: len(l.waiters[LaneInteractive]), MaxActive: longMax, QueueSize: l.cfg.InteractiveQueue},
		LaneBackground:  {Active: l.active[LaneBackground], Queued: len(l.waiters[LaneBackground]), MaxActive: bgMax, QueueSize: l.cfg.BackgroundQueue},
	}
}
