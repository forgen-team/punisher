// Package claudecli runs the locally installed, logged-in Claude Code CLI
// (`claude -p`) for every generation. It is a port of the claude-gateway
// runner: tools are disabled, the prompt goes to stdin, each call runs in an
// empty temp dir, the whole process group is killed on timeout, and a
// two-lane limiter serves interactive calls before background ones.
package claudecli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/llm"
)

// Config sizes the client. Zero values get sane defaults.
type Config struct {
	// Bin is the claude executable (path or name). "" = auto-detect.
	Bin string
	// TempBase holds per-call empty working directories.
	TempBase string
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
	DefaultModel          string        // default "sonnet"
	DefaultTimeout        time.Duration // default 180s
	MaxTimeout            time.Duration // default 600s
	// DailyLimit caps CLI calls per local day; 0 = unlimited (default).
	DailyLimit int
}

func (c Config) normalized() Config {
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
	if c.DefaultModel == "" {
		c.DefaultModel = "sonnet"
	}
	if c.DefaultTimeout <= 0 {
		c.DefaultTimeout = 180 * time.Second
	}
	if c.MaxTimeout < c.DefaultTimeout {
		c.MaxTimeout = 600 * time.Second
	}
	if c.TempBase == "" {
		c.TempBase = filepath.Join(os.TempDir(), "punisher-claude")
	}
	return c
}

// Client implements the services.LLM interface on top of the local CLI.
type Client struct {
	cfg     Config
	limiter *Limiter

	mu          sync.Mutex
	runner      *Runner
	lastFailure *Failure
	dayKey      string
	dayCalls    int
	auth        authCache
}

// Failure is the most recent auth/rate failure seen on a real call.
type Failure struct {
	Code string    `json:"code"`
	At   time.Time `json:"at"`
}

type authCache struct {
	at       time.Time
	bin      string
	loggedIn *bool
}

// New builds a client. It does not fail when claude is missing: calls (and
// Status) report claude_unavailable until the CLI is installed.
func New(cfg Config) *Client {
	cfg = cfg.normalized()
	return &Client{cfg: cfg, limiter: NewLimiter(LimiterConfig{
		Total: cfg.Concurrency, BackgroundMax: cfg.BackgroundConcurrency,
		InteractiveQueue: cfg.InteractiveQueue, BackgroundQueue: cfg.BackgroundQueue, ShortQueue: cfg.ShortQueue,
	})}
}

// ResolveBin finds the claude executable: an explicit path/name first, then
// PATH, then the usual install locations (a GUI-launched process may have a
// minimal PATH). Returns "" when nothing is found.
func ResolveBin(explicit string) string {
	if explicit != "" {
		if p, err := exec.LookPath(explicit); err == nil {
			return p
		}
		return ""
	}
	if p, err := exec.LookPath("claude"); err == nil {
		return p
	}
	home, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(home, ".local", "bin", "claude"),
		filepath.Join(home, ".claude", "local", "claude"),
		"/opt/homebrew/bin/claude",
		"/usr/local/bin/claude",
		filepath.Join(home, ".npm-global", "bin", "claude"),
		filepath.Join(home, ".volta", "bin", "claude"),
		filepath.Join(home, "bin", "claude"),
	}
	if matches, err := filepath.Glob(filepath.Join(home, ".nvm", "versions", "node", "*", "bin", "claude")); err == nil {
		candidates = append(candidates, matches...)
	}
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return c
		}
	}
	return ""
}

// runnerFor returns the cached runner, (re)detecting the CLI when it was not
// found before or has been removed since.
func (c *Client) runnerFor(ctx context.Context) (*Runner, *APIError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.runner != nil {
		if _, err := os.Stat(c.runner.Bin); err == nil {
			return c.runner, nil
		}
		c.runner = nil
	}
	bin := ResolveBin(c.cfg.Bin)
	if bin == "" {
		return nil, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, "claude CLI not found on PATH")
	}
	r, err := NewRunner(ctx, bin, c.cfg.TempBase)
	if err != nil {
		return nil, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, err.Error())
	}
	c.runner = r
	return r, nil
}

// Create runs one non-streaming call (`--output-format json`). apiKey is ignored.
func (c *Client) Create(ctx context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	return c.call(ctx, req, nil)
}

// Stream runs one streaming call (`--output-format stream-json`), passing text
// deltas to onDelta as they arrive. apiKey is ignored.
func (c *Client) Stream(ctx context.Context, _ string, req llm.MessagesRequest, onDelta func(string) error) (llm.MessagesResult, error) {
	if onDelta == nil {
		onDelta = func(string) error { return nil }
	}
	return c.call(ctx, req, onDelta)
}

func (c *Client) call(ctx context.Context, req llm.MessagesRequest, onDelta func(string) error) (llm.MessagesResult, error) {
	start := time.Now()
	prompt := promptFrom(req)
	if strings.TrimSpace(prompt) == "" {
		return llm.MessagesResult{}, toLLMError(apiErr(http.StatusBadRequest, CodeInvalid, "empty prompt"))
	}
	model := strings.TrimSpace(req.Model)
	if model == "" {
		model = c.cfg.DefaultModel
	}
	if !modelPattern.MatchString(model) {
		return llm.MessagesResult{}, toLLMError(apiErr(http.StatusBadRequest, CodeInvalid, "model must be sonnet|haiku|opus or a claude-* id"))
	}
	runner, aerr := c.runnerFor(ctx)
	if aerr != nil {
		return llm.MessagesResult{}, toLLMError(aerr)
	}
	timeout := c.cfg.DefaultTimeout
	if req.Timeout > 0 {
		timeout = req.Timeout
	}
	if timeout > c.cfg.MaxTimeout {
		timeout = c.cfg.MaxTimeout
	}
	priority := req.Priority
	if priority == "" {
		priority = llm.PriorityFromContext(ctx)
	}
	lane := ParseLane(string(priority))

	runCtx, cancel, release, aerr := c.admit(ctx, lane, timeout)
	if aerr != nil {
		return llm.MessagesResult{}, toLLMError(aerr)
	}
	defer cancel()
	defer release()

	cliReq := Request{System: req.System, Prompt: prompt}
	var res Result
	if onDelta == nil {
		stdout, stderr, runErr := runner.Run(runCtx, runner.Args(cliReq, model, false), prompt, nil)
		res, aerr = finishRun(runCtx, ctx, runErr, stderr, func() (Result, error) { return ParseJSONOutput(stdout, model) })
	} else {
		parser := &StreamParser{FallbackModel: model, OnDelta: onDelta}
		tail, stderr, runErr := runner.Run(runCtx, runner.Args(cliReq, model, true), prompt, parser.Feed)
		res, aerr = finishRun(runCtx, ctx, runErr, stderr, func() (Result, error) { return parser.Finish(string(tail)) })
	}
	c.recordOutcome(aerr)
	if aerr != nil {
		return llm.MessagesResult{}, toLLMError(aerr)
	}
	out := llm.MessagesResult{Text: res.Text, Model: res.Model, StopReason: "end_turn", LatencyMs: time.Since(start).Milliseconds()}
	if res.Usage != nil {
		out.Usage = llm.Usage{
			InputTokens:  res.Usage.InputTokens + res.Usage.CacheReadInputTokens + res.Usage.CacheCreationInputTokens,
			OutputTokens: res.Usage.OutputTokens,
		}
	}
	return out, nil
}

// promptFrom flattens a multi-turn request into one stdin prompt.
func promptFrom(req llm.MessagesRequest) string {
	if len(req.Messages) == 0 {
		return req.User
	}
	var b strings.Builder
	for _, m := range req.Messages {
		fmt.Fprintf(&b, "[%s]\n%s\n\n", m.Role, m.Content)
	}
	if req.User != "" {
		fmt.Fprintf(&b, "[user]\n%s\n", req.User)
	}
	return strings.TrimSpace(b.String())
}

// admit waits for a lane slot and applies the optional daily cap.
//   - short / interactive: queue wait counts against the call timeout and is capped by
//     InteractiveQueueWait, so a user call fails fast behind other work;
//   - background: queue wait (BackgroundQueueWait) is separate and the call
//     timeout starts once a slot is granted.
func (c *Client) admit(parent context.Context, lane Lane, timeout time.Duration) (context.Context, context.CancelFunc, func(), *APIError) {
	wait := c.cfg.InteractiveQueueWait
	var runCtx context.Context
	var runCancel context.CancelFunc
	queueParent := parent
	if lane == LaneBackground {
		wait = c.cfg.BackgroundQueueWait
	} else {
		runCtx, runCancel = context.WithTimeout(parent, timeout)
		queueParent = runCtx
	}
	fail := func(e *APIError) (context.Context, context.CancelFunc, func(), *APIError) {
		if runCancel != nil {
			runCancel()
		}
		return nil, nil, nil, e
	}
	qctx, qcancel := context.WithTimeout(queueParent, wait)
	release, err := c.limiter.Acquire(qctx, lane)
	qcancel()
	if errors.Is(err, ErrQueueFull) {
		return fail(apiErr(http.StatusTooManyRequests, CodeQueueFull, fmt.Sprintf("%s queue is full", lane)))
	}
	if err != nil {
		if parent.Err() != nil {
			return fail(apiErr(499, CodeClientGone, "caller gave up while queued"))
		}
		return fail(apiErr(http.StatusGatewayTimeout, CodeTimeout, fmt.Sprintf("timed out waiting in %s queue", lane)))
	}
	if !c.reserveDaily() {
		release()
		return fail(apiErr(http.StatusTooManyRequests, CodeDailyCap, fmt.Sprintf("daily call cap (%d) reached", c.cfg.DailyLimit)))
	}
	if runCtx == nil {
		runCtx, runCancel = context.WithTimeout(parent, timeout)
	}
	return runCtx, runCancel, release, nil
}

func (c *Client) reserveDaily() bool {
	if c.cfg.DailyLimit <= 0 {
		return true
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	key := time.Now().Format("2006-01-02")
	if c.dayKey != key {
		c.dayKey, c.dayCalls = key, 0
	}
	if c.dayCalls >= c.cfg.DailyLimit {
		return false
	}
	c.dayCalls++
	return true
}

// finishRun turns a process outcome into a Result or APIError. A parsed CLI
// result (even on non-zero exit) is preferred because it carries the reason.
func finishRun(ctx, callerCtx context.Context, runErr error, stderr string, parse func() (Result, error)) (Result, *APIError) {
	if callerCtx.Err() != nil {
		return Result{}, apiErr(499, CodeClientGone, "caller gave up")
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return Result{}, apiErr(http.StatusGatewayTimeout, CodeTimeout, "claude call exceeded timeout; process group killed")
	}
	var execErr *exec.Error
	if errors.As(runErr, &execErr) {
		return Result{}, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, "claude executable not runnable: "+execErr.Error())
	}
	var pathErr *os.PathError
	if errors.As(runErr, &pathErr) {
		return Result{}, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, "claude executable not runnable: "+pathErr.Error())
	}
	res, err := parse()
	if err == nil && runErr == nil {
		return res, nil
	}
	var ae *APIError
	if err != nil && errors.As(err, &ae) && (ae.Code != CodeCLIError || runErr == nil) {
		return Result{}, ae
	}
	detail := strings.TrimSpace(stderr)
	if detail == "" && err != nil {
		detail = err.Error()
	}
	if detail == "" && runErr != nil {
		detail = runErr.Error()
	}
	return Result{}, classifyFailure(detail)
}

func (c *Client) recordOutcome(aerr *APIError) {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case aerr == nil:
		c.lastFailure = nil
	case aerr.Code == CodeNotLoggedIn || aerr.Code == CodeRateLimited || aerr.Code == CodeCLIMissing:
		c.lastFailure = &Failure{Code: aerr.Code, At: time.Now()}
		if aerr.Code == CodeNotLoggedIn {
			c.auth = authCache{} // re-check login on the next status call
		}
	}
}

// toLLMError maps CLI error codes onto the llm sentinel errors.
func toLLMError(ae *APIError) error {
	if ae.Status == 499 {
		return context.Canceled
	}
	sentinel := llm.ErrServerError
	switch ae.Code {
	case CodeNotLoggedIn, CodeCLIMissing:
		sentinel = llm.ErrGenerationUnavailable
	case CodeRateLimited, CodeQueueFull, CodeDailyCap:
		sentinel = llm.ErrRateLimited
	case CodeTimeout:
		sentinel = llm.ErrTimeout
	case CodeInvalid:
		sentinel = llm.ErrInvalidRequest
	}
	return &llm.LLMError{StatusCode: ae.Status, Code: ae.Code, Message: "claude: " + ae.Code + ": " + ae.Message, Err: sentinel}
}

// ErrorCode returns the CLI error code carried by err ("" if none).
func ErrorCode(err error) string {
	var le *llm.LLMError
	if errors.As(err, &le) {
		return le.Code
	}
	return ""
}

// Status is the CLI readiness shown in the UI.
type Status struct {
	// Ready is true when the CLI is installed and not known to be logged out.
	Ready     bool   `json:"ready"`
	Installed bool   `json:"installed"`
	Version   string `json:"version,omitempty"`
	// LoggedIn is nil when the CLI cannot report it (older versions).
	LoggedIn    *bool    `json:"loggedIn"`
	Code        string   `json:"code,omitempty"`
	LastFailure *Failure `json:"lastFailure,omitempty"`
	Active      int      `json:"active"`
	Queued      int      `json:"queued"`
}

const authCacheTTL = 30 * time.Second

// Status reports whether the CLI is installed and logged in. It runs
// `claude --version` and `claude auth status --json` (cached for 30s); it
// never makes a model call.
func (c *Client) Status(ctx context.Context) Status {
	var st Status
	st.Active, st.Queued = c.limiter.Stats()
	runner, aerr := c.runnerFor(ctx)
	if aerr != nil {
		st.Code = CodeCLIMissing
		return st
	}
	st.Installed = true
	version, err := runner.Version(ctx)
	if err != nil {
		st.Code = CodeCLIMissing
		return st
	}
	st.Version = version
	st.LoggedIn = c.loggedIn(ctx, runner)

	c.mu.Lock()
	st.LastFailure = c.lastFailure
	c.mu.Unlock()
	switch {
	case st.LoggedIn != nil && !*st.LoggedIn:
		st.Code = CodeNotLoggedIn
	case st.LastFailure != nil && st.LastFailure.Code == CodeNotLoggedIn && st.LoggedIn == nil:
		st.Code = CodeNotLoggedIn
	case st.LastFailure != nil && st.LastFailure.Code == CodeRateLimited && time.Since(st.LastFailure.At) < 15*time.Minute:
		st.Code = CodeRateLimited
	}
	st.Ready = st.Code == "" || st.Code == CodeRateLimited
	return st
}

func (c *Client) loggedIn(ctx context.Context, runner *Runner) *bool {
	c.mu.Lock()
	cached := c.auth
	c.mu.Unlock()
	if cached.bin == runner.Bin && time.Since(cached.at) < authCacheTTL {
		return cached.loggedIn
	}
	var result *bool
	out, err := runner.output(ctx, 15*time.Second, "auth", "status", "--json")
	if err == nil || out != "" {
		var parsed struct {
			LoggedIn *bool `json:"loggedIn"`
		}
		if json.Unmarshal([]byte(strings.TrimSpace(out)), &parsed) == nil {
			result = parsed.LoggedIn
		}
	}
	c.mu.Lock()
	c.auth = authCache{at: time.Now(), bin: runner.Bin, loggedIn: result}
	c.mu.Unlock()
	return result
}
