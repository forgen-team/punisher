// Package agycli runs the locally installed, logged-in Antigravity CLI
// (`agy -p`, headless mode) for generation — the supported way to use Gemini
// with a Google subscription. Built from the official headless docs
// (antigravity.google/docs/cli/headless and /cli/reference): the prompt is
// the -p argument, `--output-format json` returns one envelope and
// `stream-json` NDJSON events. Optional flags are probed with `agy --help`
// and skipped when not listed. Each call runs in an empty temp dir and the
// whole process group is killed on timeout; admission is the shared
// clicore.Gate.
package agycli

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/clicore"
	"github.com/forgen-team/punisher/internal/llm"
)

// Error codes of this provider.
const (
	CodeCLIMissing  = "agy_unavailable"
	CodeNotLoggedIn = "agy_not_logged_in"
	CodeRateLimited = "agy_rate_limited"
	CodeTimeout     = "agy_timeout"
	CodeCLIError    = "agy_error"
)

var codes = clicore.CodesFor("agy", "run `agy` once in a terminal to sign in")

// DefaultModelName is recorded as the model when none is configured (the CLI's default).
const DefaultModelName = "agy"

// Config sizes the client.
type Config struct {
	// Bin is the agy executable (path or name). "" = auto-detect.
	Bin string
	// TempBase holds per-call empty working directories.
	TempBase string
	// Gate is the shared admission gate; nil = a private default gate.
	Gate *clicore.Gate
}

// Client implements services.LLM on top of `agy -p`.
type Client struct {
	cfg     Config
	gate    *clicore.Gate
	runners *clicore.BinCache[*runner]
	fails   clicore.Tracker
	status  clicore.StatusCache
}

// New builds a client. It does not fail when agy is missing: calls (and
// Status) report agy_unavailable until the CLI is installed.
func New(cfg Config) *Client {
	if cfg.TempBase == "" {
		cfg.TempBase = filepath.Join(os.TempDir(), "punisher-agy")
	}
	gate := cfg.Gate
	if gate == nil {
		gate = clicore.NewGate(clicore.GateConfig{})
	}
	return &Client{cfg: cfg, gate: gate, runners: &clicore.BinCache[*runner]{}}
}

// ResolveBin finds the agy executable ("" when not installed).
func ResolveBin(explicit string) string {
	return clicore.ResolveBin(explicit, "agy", clicore.CommonBinCandidates("agy"))
}

// optionalFlags are dropped when `agy --help` does not list them.
var optionalFlags = []string{"--output-format", "--model", "--print-timeout", "--sandbox", "--disable-slash-commands"}

type runner struct {
	clicore.Proc
	supported map[string]bool
	probeErr  error // `agy --help` could not be read
}

func newRunner(ctx context.Context, bin, tempBase string) (*runner, bool, error) {
	proc, err := clicore.NewProc(bin, tempBase, []string{"NO_COLOR=1"}, clicore.BaseEnvKeys...)
	if err != nil {
		return nil, false, err
	}
	r := &runner{Proc: proc}
	// The first caller's ctx may be cancelled any moment; the probe result is shared.
	r.supported, r.probeErr = proc.ProbeHelp(context.WithoutCancel(ctx), optionalFlags, "--help")
	return r, r.probeErr == nil, nil
}

func (r *runner) supports(flag string) bool { return r.probeErr == nil && r.supported[flag] }

// probeError refuses a call when help could not be read: the flags (and
// whether the output is JSON at all) are unknown, so nothing is assumed;
// the probe is retried shortly (clicore.BinCache).
func (r *runner) probeError() *clicore.APIError {
	if r.probeErr == nil {
		return nil
	}
	return clicore.NewError(http.StatusBadGateway, CodeCLIError, "could not read `agy --help`; check the agy CLI ("+r.probeErr.Error()+")")
}

// modelPattern accepts slugs like gemini-3.8-flash-medium.
var modelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:\-]{0,79}$`)

// Args builds the headless arguments. The prompt is the -p value: the docs
// only show it as an argument and reading the prompt from stdin is
// unverified, so it stays argv — capped at clicore.MaxArgPromptBytes (call)
// with E2BIG reported as invalid_request (clicore.Finish). stdin stays empty.
func (r *runner) Args(prompt, model string, stream bool, timeout time.Duration) []string {
	args := []string{"-p", prompt}
	if r.supports("--output-format") {
		format := "json"
		if stream {
			format = "stream-json"
		}
		args = append(args, "--output-format", format)
	}
	if model != "" && r.supports("--model") {
		args = append(args, "--model", model)
	}
	if timeout > 0 && r.supports("--print-timeout") {
		args = append(args, "--print-timeout", fmt.Sprintf("%ds", int(timeout.Seconds())))
	}
	if r.supports("--sandbox") {
		args = append(args, "--sandbox")
	}
	if r.supports("--disable-slash-commands") {
		args = append(args, "--disable-slash-commands")
	}
	return args
}

func (c *Client) runnerFor(ctx context.Context) (*runner, *clicore.APIError) {
	r, err := c.runners.Get(func() string { return ResolveBin(c.cfg.Bin) },
		func(bin string) (*runner, bool, error) { return newRunner(ctx, bin, c.cfg.TempBase) })
	if clicore.IsBinNotFound(err) {
		return nil, codes.MissingError("agy CLI not found on PATH")
	}
	if err != nil {
		return nil, codes.MissingError(err.Error())
	}
	return r, nil
}

// noToolsInstruction keeps the agent answering in text: Punisher only reads the response.
const noToolsInstruction = "Answer directly in your response. Do not run commands, read or edit files, or browse."

// Create runs one call (`--output-format json`). apiKey is ignored.
func (c *Client) Create(ctx context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	return c.call(ctx, req, nil)
}

// Stream runs one call (`--output-format stream-json`), passing text_delta
// fragments to onDelta. apiKey is ignored.
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
		return llm.MessagesResult{}, codes.ToLLMError(clicore.NewError(http.StatusBadRequest, clicore.CodeInvalid, "empty prompt"))
	}
	model := strings.TrimSpace(req.Model)
	if model != "" && !modelPattern.MatchString(model) {
		return llm.MessagesResult{}, codes.ToLLMError(clicore.NewError(http.StatusBadRequest, clicore.CodeInvalid, "invalid agy model slug"))
	}
	full := clicore.WithSystem(strings.TrimSpace(req.System+"\n\n"+noToolsInstruction), prompt)
	if len(full) > clicore.MaxArgPromptBytes {
		return llm.MessagesResult{}, codes.ToLLMError(clicore.NewError(http.StatusBadRequest, clicore.CodeInvalid,
			fmt.Sprintf("prompt is %d bytes; agy takes the prompt as a command-line argument (max %d)", len(full), clicore.MaxArgPromptBytes)))
	}
	r, aerr := c.runnerFor(ctx)
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	if aerr := r.probeError(); aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	runCtx, done, aerr := c.gate.Admit(ctx, req, codes)
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	defer done()

	args := r.Args(full, model, onDelta != nil, c.gate.Timeout(req))
	var res Result
	if onDelta == nil {
		stdout, stderr, runErr := r.Run(runCtx, args, "", nil)
		res, aerr = clicore.Finish(runCtx, ctx, codes, runErr, stderr, func() (Result, error) { return ParseJSON(stdout) })
	} else {
		parser := &StreamParser{OnDelta: onDelta}
		_, stderr, runErr := r.Run(runCtx, args, "", parser.Feed)
		res, aerr = clicore.Finish(runCtx, ctx, codes, runErr, stderr, parser.Finish)
	}
	c.fails.Record(aerr, codes)
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	name := res.Model
	if name == "" {
		name = model
	}
	if name == "" {
		name = DefaultModelName
	}
	return llm.MessagesResult{Text: res.Text, Model: name, StopReason: "end_turn", LatencyMs: time.Since(start).Milliseconds(),
		Usage: llm.Usage{InputTokens: res.InputTokens, OutputTokens: res.OutputTokens}}, nil
}

// statusTTL is how long Status reuses `agy --version` (the UI polls /v1/status).
const statusTTL = 30 * time.Second

// Status reports install state (`agy --version`, cached for statusTTL). The
// docs list no login status command, so LoggedIn stays nil and a failed
// call's authentication error is what marks the CLI logged out.
func (c *Client) Status(ctx context.Context) clicore.Status {
	var st clicore.Status
	st.Active, st.Queued = c.gate.Limiter().Stats()
	r, aerr := c.runnerFor(ctx)
	if aerr != nil {
		st.Code = CodeCLIMissing
		return st
	}
	st.Installed = true
	probe, ok := c.status.Get(r.Bin, statusTTL)
	if !ok {
		if out, err := r.Output(ctx, 10*time.Second, "--version"); err == nil {
			probe.Version = strings.TrimSpace(out)
		}
		if ctx.Err() == nil { // a cut-short probe is not remembered
			c.status.Put(r.Bin, probe)
		}
	}
	st.Version = probe.Version
	c.fails.Finish(&st, codes)
	return st
}
