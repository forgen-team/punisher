package claudecli

import (
	"context"
	"errors"
	"strings"
	"sync"
)

// ErrQueueFull is returned when a lane's wait queue is full.
var ErrQueueFull = errors.New("queue full")

// Lane is a scheduling class. Interactive = user-initiated calls; background
// = scheduler/worker/pipeline calls.
type Lane string

const (
	LaneInteractive Lane = "interactive"
	LaneBackground  Lane = "background"
)

// ParseLane maps a request priority to a lane; anything but "background" is
// interactive so an unknown/empty value can never demote a user call.
func ParseLane(raw string) Lane {
	if strings.EqualFold(strings.TrimSpace(raw), string(LaneBackground)) {
		return LaneBackground
	}
	return LaneInteractive
}

// LimiterConfig sizes the two lanes.
type LimiterConfig struct {
	Total            int // total parallel CLI processes (both lanes)
	BackgroundMax    int // max parallel background processes
	InteractiveQueue int // interactive waiters before 429
	BackgroundQueue  int // background waiters before 429
}

// Limiter is a two-lane semaphore:
//   - interactive waiters are served FIFO and always before background;
//   - background holds at most BackgroundMax slots and, when Total >= 2,
//     never takes the last free slot, so one slot is always available for an
//     interactive call even while background work runs;
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
	return &Limiter{cfg: cfg, active: map[Lane]int{}, waiters: map[Lane][]*waiter{}}
}

// reserved is the number of slots background may never use.
func (l *Limiter) reserved() int {
	if l.cfg.Total >= 2 {
		return 1
	}
	return 0
}

func (l *Limiter) totalActive() int { return l.active[LaneInteractive] + l.active[LaneBackground] }

// canGrant reports whether the head waiter of lane may start now. Caller holds mu.
func (l *Limiter) canGrant(lane Lane) bool {
	if l.totalActive() >= l.cfg.Total {
		return false
	}
	if lane == LaneInteractive {
		return true
	}
	return len(l.waiters[LaneInteractive]) == 0 &&
		l.active[LaneBackground] < l.cfg.BackgroundMax &&
		l.totalActive() < l.cfg.Total-l.reserved()
}

// dispatch grants slots to queue heads, interactive first. Caller holds mu.
func (l *Limiter) dispatch() {
	for _, lane := range []Lane{LaneInteractive, LaneBackground} {
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
	if lane == LaneBackground {
		return l.cfg.BackgroundQueue
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
	// A cancelled interactive waiter may have been blocking background.
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
	return l.totalActive(), len(l.waiters[LaneInteractive]) + len(l.waiters[LaneBackground])
}

// LaneStats returns per-lane counts.
func (l *Limiter) LaneStats() map[Lane]LaneStats {
	l.mu.Lock()
	defer l.mu.Unlock()
	bgMax := l.cfg.BackgroundMax
	if limit := l.cfg.Total - l.reserved(); bgMax > limit {
		bgMax = limit
	}
	return map[Lane]LaneStats{
		LaneInteractive: {Active: l.active[LaneInteractive], Queued: len(l.waiters[LaneInteractive]), MaxActive: l.cfg.Total, QueueSize: l.cfg.InteractiveQueue},
		LaneBackground:  {Active: l.active[LaneBackground], Queued: len(l.waiters[LaneBackground]), MaxActive: bgMax, QueueSize: l.cfg.BackgroundQueue},
	}
}
