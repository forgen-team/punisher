package clicore

import (
	"sync"
	"time"
)

// Failure is the most recent auth/rate/missing failure seen on a real call.
type Failure struct {
	Code string    `json:"code"`
	At   time.Time `json:"at"`
}

// Status is a CLI's readiness shown in the UI.
type Status struct {
	// Ready is true when the CLI is installed and not known to be logged out.
	Ready     bool   `json:"ready"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	// LoggedIn is nil when the CLI cannot report it.
	LoggedIn    *bool    `json:"loggedIn"`
	Code        string   `json:"code,omitempty"`
	LastFailure *Failure `json:"lastFailure,omitempty"`
	Active      int      `json:"active"`
	Queued      int      `json:"queued"`
}

// rateLimitMemory is how long a rate-limit failure is shown as the status code.
const rateLimitMemory = 15 * time.Minute

// Tracker remembers the last auth/rate/missing failure of a client.
type Tracker struct {
	mu   sync.Mutex
	last *Failure
}

// Record notes a call outcome: success clears the failure. It reports
// whether the call failed because the CLI is not logged in (callers then
// drop their cached login state).
func (t *Tracker) Record(aerr *APIError, codes Codes) (loggedOut bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	switch {
	case aerr == nil:
		t.last = nil
	case aerr.Code == codes.NotLoggedIn || aerr.Code == codes.RateLimited || aerr.Code == codes.Missing:
		t.last = &Failure{Code: aerr.Code, At: time.Now()}
		return aerr.Code == codes.NotLoggedIn
	}
	return false
}

// Last is the remembered failure (nil = none).
func (t *Tracker) Last() *Failure {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last
}

// Finish fills Code/Ready of an installed CLI's status from its login state
// and the remembered failure.
func (t *Tracker) Finish(st *Status, codes Codes) {
	st.LastFailure = t.Last()
	switch {
	case st.LoggedIn != nil && !*st.LoggedIn:
		st.Code = codes.NotLoggedIn
	case st.LastFailure != nil && st.LastFailure.Code == codes.NotLoggedIn && st.LoggedIn == nil:
		st.Code = codes.NotLoggedIn
	case st.LastFailure != nil && st.LastFailure.Code == codes.RateLimited && time.Since(st.LastFailure.At) < rateLimitMemory:
		st.Code = codes.RateLimited
	}
	st.Ready = st.Code == "" || st.Code == codes.RateLimited
}

// StatusProbe is what a status call learned by running the CLI.
type StatusProbe struct {
	Version  string
	LoggedIn *bool
	// Broken: `--version` failed (the CLI is installed but not runnable).
	Broken bool
}

// StatusCache keeps a client's last StatusProbe for a short while so a
// polled status endpoint does not spawn `--version` / login checks every time.
type StatusCache struct {
	mu    sync.Mutex
	bin   string
	at    time.Time
	probe StatusProbe
}

// Get returns the probe of bin when it is younger than ttl.
func (c *StatusCache) Get(bin string, ttl time.Duration) (StatusProbe, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.at.IsZero() || c.bin != bin || time.Since(c.at) >= ttl {
		return StatusProbe{}, false
	}
	return c.probe, true
}

// Put remembers the probe of bin.
func (c *StatusCache) Put(bin string, p StatusProbe) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.bin, c.at, c.probe = bin, time.Now(), p
}

// Reset forgets the probe (e.g. after a call failed as logged out).
func (c *StatusCache) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.at = time.Time{}
}
