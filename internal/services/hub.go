package services

import (
	"sync"
	"time"
)

// Stream event types sent over SSE.
const (
	EventDelta  = "delta"  // {"text": "..."}
	EventStatus = "status" // {"stage": "generating" | "validating"}
	EventDone   = "done"   // {"lectureId", "status", "flagged", "issues"}
	EventError  = "error"  // {"code", "message"}
)

// Event is one stream event.
type Event struct {
	Type string
	Data any
}

type jobStream struct {
	target string
	events []Event
	subs   map[chan Event]struct{}
	closed bool
}

// Hub fans out instant-lecture generation events to any number of SSE
// subscribers, replaying past events to late joiners. It also guarantees one
// in-process generation per target (objective set + depth); the DB partial
// unique index guards across processes.
type Hub struct {
	mu      sync.Mutex
	streams map[string]*jobStream
	targets map[string]string // target → jobID while running
	// Linger keeps finished streams for late subscribers.
	Linger time.Duration
}

// NewHub returns an empty hub.
func NewHub() *Hub {
	return &Hub{streams: map[string]*jobStream{}, targets: map[string]string{}, Linger: 2 * time.Minute}
}

// ActiveJob returns the running job for a target, if any.
func (h *Hub) ActiveJob(target string) (string, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	id, ok := h.targets[target]
	return id, ok
}

// Start runs fn in a goroutine for jobID unless target already has a running
// job, in which case that job id is returned with started=false.
func (h *Hub) Start(jobID, target string, fn func(emit func(Event))) (string, bool) {
	h.mu.Lock()
	if existing, ok := h.targets[target]; ok {
		h.mu.Unlock()
		return existing, false
	}
	h.targets[target] = jobID
	h.streams[jobID] = &jobStream{target: target, subs: map[chan Event]struct{}{}}
	h.mu.Unlock()

	go func() {
		defer h.finish(jobID)
		fn(func(e Event) { h.publish(jobID, e) })
	}()
	return jobID, true
}

func (h *Hub) publish(jobID string, e Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	st, ok := h.streams[jobID]
	if !ok || st.closed {
		return
	}
	st.events = append(st.events, e)
	for ch := range st.subs {
		select {
		case ch <- e:
		default:
			// Slow subscriber: drop it; it can reconnect and get a full replay.
			delete(st.subs, ch)
			close(ch)
		}
	}
}

func (h *Hub) finish(jobID string) {
	h.mu.Lock()
	st, ok := h.streams[jobID]
	if ok {
		st.closed = true
		for ch := range st.subs {
			close(ch)
		}
		st.subs = nil
		if h.targets[st.target] == jobID {
			delete(h.targets, st.target)
		}
	}
	linger := h.Linger
	h.mu.Unlock()
	if ok {
		time.AfterFunc(linger, func() {
			h.mu.Lock()
			delete(h.streams, jobID)
			h.mu.Unlock()
		})
	}
}

// Subscribe returns past events and a channel of future ones (closed when the
// job ends). ok=false if the hub does not know the job (finished long ago or
// another process) — callers then fall back to the DB job status.
func (h *Hub) Subscribe(jobID string) (replay []Event, ch <-chan Event, cancel func(), ok bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	st, found := h.streams[jobID]
	if !found {
		return nil, nil, func() {}, false
	}
	replay = append([]Event(nil), st.events...)
	c := make(chan Event, 512)
	if st.closed {
		close(c)
		return replay, c, func() {}, true
	}
	st.subs[c] = struct{}{}
	cancel = func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, live := st.subs[c]; live {
			delete(st.subs, c)
			close(c)
		}
	}
	return replay, c, cancel, true
}
