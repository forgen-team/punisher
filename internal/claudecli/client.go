// Package claudecli runs the locally installed, logged-in Claude Code CLI
// (`claude -p`) for every generation. It is a port of the claude-gateway
// runner: tools are disabled, the prompt goes to stdin, each call runs in an
// empty temp dir, the whole process group is killed on timeout, and a
// two-lane limiter serves interactive calls before background ones.
package claudecli

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/clicore"
	"github.com/forgen-team/punisher/internal/llm"
)

// Config sizes the client. Zero values get sane defaults.
type Config struct {
	// Bin is the claude executable (path or name). "" = auto-detect.
	Bin string
	// TempBase holds per-call empty working directories.
	TempBase string
	// Gate is the shared admission gate (lanes, timeouts, daily cap). nil =
	// a private gate built from the fields below.
	Gate *clicore.Gate
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

// GateConfig is the admission part of the config.
func (c Config) GateConfig() clicore.GateConfig {
	return clicore.GateConfig{Concurrency: c.Concurrency, BackgroundConcurrency: c.BackgroundConcurrency,
		InteractiveQueue: c.InteractiveQueue, ShortQueue: c.ShortQueue, BackgroundQueue: c.BackgroundQueue,
		InteractiveQueueWait: c.InteractiveQueueWait, BackgroundQueueWait: c.BackgroundQueueWait,
		DefaultTimeout: c.DefaultTimeout, MaxTimeout: c.MaxTimeout, DailyLimit: c.DailyLimit}
}

// codes are claude's stable failure codes.
var codes = clicore.CodesFor("claude", "run `claude` once in a terminal to log in")

// Client implements the services.LLM interface on top of the local CLI.
type Client struct {
	cfg     Config
	gate    *clicore.Gate
	limiter *Limiter
	runners *clicore.BinCache[*Runner]
	fails   clicore.Tracker

	mu      sync.Mutex
	auth    authCache
	version clicore.StatusCache // `claude --version`, reused for authCacheTTL
}

// Failure is the most recent auth/rate failure seen on a real call.
type Failure = clicore.Failure

type authCache struct {
	at       time.Time
	bin      string
	loggedIn *bool
}

// New builds a client. It does not fail when claude is missing: calls (and
// Status) report claude_unavailable until the CLI is installed.
func New(cfg Config) *Client {
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = "sonnet"
	}
	if cfg.TempBase == "" {
		cfg.TempBase = filepath.Join(os.TempDir(), "punisher-claude")
	}
	gate := cfg.Gate
	if gate == nil {
		gate = clicore.NewGate(cfg.GateConfig())
	}
	return &Client{cfg: cfg, gate: gate, limiter: gate.Limiter(), runners: &clicore.BinCache[*Runner]{}}
}

// ResolveBin finds the claude executable: an explicit path/name first, then
// PATH, then the usual install locations (a GUI-launched process may have a
// minimal PATH). Returns "" when nothing is found.
func ResolveBin(explicit string) string {
	home, _ := os.UserHomeDir()
	common := clicore.CommonBinCandidates("claude")
	candidates := append([]string{common[0], filepath.Join(home, ".claude", "local", "claude")}, common[1:]...)
	return clicore.ResolveBin(explicit, "claude", candidates)
}

// runnerFor returns the cached runner, (re)detecting the CLI when it was not
// found before or has been removed since.
func (c *Client) runnerFor(ctx context.Context) (*Runner, *APIError) {
	r, err := c.runners.Get(func() string { return ResolveBin(c.cfg.Bin) },
		func(bin string) (*Runner, bool, error) {
			r, err := NewRunner(ctx, bin, c.cfg.TempBase)
			return r, err == nil && r.probeErr == nil, err
		})
	if clicore.IsBinNotFound(err) {
		return nil, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, "claude CLI not found on PATH")
	}
	if err != nil {
		return nil, apiErr(http.StatusServiceUnavailable, CodeCLIMissing, err.Error())
	}
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
	prompt := clicore.PromptFrom(req)
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
	runCtx, done, aerr := c.gate.Admit(ctx, req, codes)
	if aerr != nil {
		return llm.MessagesResult{}, toLLMError(aerr)
	}
	defer done()

	cliReq := Request{System: req.System, Prompt: prompt}
	var res Result
	if onDelta == nil {
		stdout, stderr, runErr := runner.Run(runCtx, runner.Args(cliReq, model, false), prompt, nil)
		res, aerr = clicore.Finish(runCtx, ctx, codes, runErr, stderr, func() (Result, error) { return ParseJSONOutput(stdout, model) })
	} else {
		parser := &StreamParser{FallbackModel: model, OnDelta: onDelta}
		tail, stderr, runErr := runner.Run(runCtx, runner.Args(cliReq, model, true), prompt, parser.Feed)
		res, aerr = clicore.Finish(runCtx, ctx, codes, runErr, stderr, func() (Result, error) { return parser.Finish(string(tail)) })
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

func (c *Client) recordOutcome(aerr *APIError) {
	if c.fails.Record(aerr, codes) {
		c.mu.Lock()
		c.auth = authCache{} // re-check login on the next status call
		c.mu.Unlock()
	}
}

// toLLMError maps CLI error codes onto the llm sentinel errors.
func toLLMError(ae *APIError) error { return codes.ToLLMError(ae) }

// ErrorCode returns the CLI error code carried by err ("" if none).
func ErrorCode(err error) string { return clicore.ErrorCode(err) }

// Status is the CLI readiness shown in the UI.
type Status = clicore.Status

const authCacheTTL = 30 * time.Second

// Status reports whether the CLI is installed and logged in. It runs
// `claude --version` and `claude auth status --json` (each cached for 30s);
// it never makes a model call.
func (c *Client) Status(ctx context.Context) Status {
	var st Status
	st.Active, st.Queued = c.limiter.Stats()
	runner, aerr := c.runnerFor(ctx)
	if aerr != nil {
		st.Code = CodeCLIMissing
		return st
	}
	st.Installed = true
	probe, ok := c.version.Get(runner.Bin, authCacheTTL)
	if !ok {
		version, err := runner.Version(ctx)
		probe = clicore.StatusProbe{Version: version, Broken: err != nil}
		if ctx.Err() == nil {
			c.version.Put(runner.Bin, probe)
		}
	}
	if probe.Broken {
		st.Code = CodeCLIMissing
		return st
	}
	st.Version = probe.Version
	st.LoggedIn = c.loggedIn(ctx, runner)
	c.fails.Finish(&st, codes)
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
	out, err := runner.Output(ctx, 15*time.Second, "auth", "status", "--json")
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
