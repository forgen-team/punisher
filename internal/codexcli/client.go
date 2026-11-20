// Package codexcli runs the locally installed, logged-in OpenAI Codex CLI
// (`codex exec`) for generation. Like claudecli: the prompt goes to stdin,
// each call runs in an empty temp dir with a read-only sandbox, the user's
// config.toml (MCP servers, profiles) and exec-policy rules are not loaded,
// the session is not persisted, and the whole process group is killed on
// timeout. Admission (lanes, timeouts, daily cap) is the shared clicore.Gate.
package codexcli

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/clicore"
	"github.com/forgen-team/punisher/internal/llm"
)

// Error codes of this provider.
const (
	CodeCLIMissing  = "codex_unavailable"
	CodeNotLoggedIn = "codex_not_logged_in"
	CodeRateLimited = "codex_rate_limited"
	CodeTimeout     = "codex_timeout"
	CodeCLIError    = "codex_error"
)

var codes = clicore.CodesFor("codex", "run `codex login` in a terminal")

// DefaultModelName is recorded as the model when none is configured (the CLI's default).
const DefaultModelName = "codex"

// Config sizes the client.
type Config struct {
	// Bin is the codex executable (path or name). "" = auto-detect.
	Bin string
	// TempBase holds per-call empty working directories.
	TempBase string
	// Gate is the shared admission gate; nil = a private default gate.
	Gate *clicore.Gate
}

// Client implements services.LLM on top of `codex exec`.
type Client struct {
	cfg     Config
	gate    *clicore.Gate
	runners *clicore.BinCache[*runner]
	fails   clicore.Tracker
	status  clicore.StatusCache
}

// New builds a client. It does not fail when codex is missing: calls (and
// Status) report codex_unavailable until the CLI is installed.
func New(cfg Config) *Client {
	if cfg.TempBase == "" {
		cfg.TempBase = filepath.Join(os.TempDir(), "punisher-codex")
	}
	gate := cfg.Gate
	if gate == nil {
		gate = clicore.NewGate(clicore.GateConfig{})
	}
	return &Client{cfg: cfg, gate: gate, runners: &clicore.BinCache[*runner]{}}
}

// ResolveBin finds the codex executable ("" when not installed).
func ResolveBin(explicit string) string {
	return clicore.ResolveBin(explicit, "codex", clicore.CommonBinCandidates("codex"))
}

// requiredFlags of `codex exec` carry the isolation: without them codex
// could write files or load the user's MCP servers and profiles, so a codex
// that does not list them (or whose help cannot be read) is refused.
var requiredFlags = []string{"--sandbox", "--ignore-user-config"}

// optionalFlags are cosmetic/hygiene flags dropped when help does not list
// them; --config and --disable gate the tool and reasoning overrides.
var optionalFlags = []string{"--skip-git-repo-check", "--ephemeral", "--ignore-rules", "--color", "--config", "--disable"}

// toolFeatures are codex feature flags turned off with --disable when
// `codex features list` lists them (an unknown name makes codex exit): with
// them off the model has no shell tool at all (verified with codex-cli
// 0.163.0: asked to run a command it answers without command_execution).
var toolFeatures = []string{"shell_tool", "unified_exec"}

type runner struct {
	clicore.Proc
	supported map[string]bool
	probeErr  error    // `codex exec --help` could not be read
	disable   []string // toolFeatures this codex knows
}

func newRunner(ctx context.Context, bin, tempBase string) (*runner, bool, error) {
	proc, err := clicore.NewProc(bin, tempBase, []string{"NO_COLOR=1"}, slices.Concat(clicore.BaseEnvKeys, []string{"CODEX_HOME"})...)
	if err != nil {
		return nil, false, err
	}
	// The first caller's ctx may be cancelled any moment; the probe result is shared.
	pctx := context.WithoutCancel(ctx)
	r := &runner{Proc: proc}
	r.supported, r.probeErr = proc.ProbeHelp(pctx, slices.Concat(requiredFlags, optionalFlags), "exec", "--help")
	if r.probeErr == nil && r.supported["--disable"] {
		r.disable = knownFeatures(pctx, proc, toolFeatures)
	}
	return r, r.probeErr == nil, nil
}

// knownFeatures returns the wanted features listed by `codex features list`
// (first column); none when it fails.
func knownFeatures(ctx context.Context, proc clicore.Proc, wanted []string) []string {
	out, err := proc.Output(ctx, 15*time.Second, "features", "list")
	if err != nil {
		return nil
	}
	listed := map[string]bool{}
	for _, line := range strings.Split(out, "\n") {
		if f := strings.Fields(line); len(f) > 0 {
			listed[f[0]] = true
		}
	}
	var known []string
	for _, w := range wanted {
		if listed[w] {
			known = append(known, w)
		}
	}
	return known
}

func (r *runner) supports(flag string) bool { return r.probeErr == nil && r.supported[flag] }

// requiredError refuses a codex that cannot be run isolated.
func (r *runner) requiredError() *clicore.APIError {
	if r.probeErr != nil {
		return clicore.NewError(http.StatusBadGateway, CodeCLIError,
			"could not read `codex exec --help`, so the read-only sandbox cannot be confirmed; check or update the codex CLI ("+r.probeErr.Error()+")")
	}
	var missing []string
	for _, f := range requiredFlags {
		if !r.supported[f] {
			missing = append(missing, f)
		}
	}
	if len(missing) > 0 {
		return clicore.NewError(http.StatusBadGateway, CodeCLIError,
			"this codex CLI lacks "+strings.Join(missing, ", ")+" (read-only sandbox / config isolation); update it: npm install -g @openai/codex or brew upgrade codex")
	}
	return nil
}

// modelPattern accepts model ids like gpt-5.5, o3, gpt-5-codex.
var modelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:\-]{0,79}$`)

// Args builds `codex exec` arguments; the prompt is read from stdin ("-").
// lowEffort asks for low reasoning effort (fast tier on the CLI's default
// model; `-c model_reasoning_effort="low"` verified with codex-cli 0.163.0).
// Call requiredError first: the required flags are always added here.
func (r *runner) Args(model string, lowEffort bool) []string {
	args := []string{"exec", "--json", "--sandbox", "read-only", "--ignore-user-config"}
	for _, f := range []string{"--skip-git-repo-check", "--ephemeral", "--ignore-rules"} {
		if r.supports(f) {
			args = append(args, f)
		}
	}
	if r.supports("--color") {
		args = append(args, "--color", "never")
	}
	for _, f := range r.disable {
		args = append(args, "--disable", f)
	}
	if r.supports("--config") {
		args = append(args, "-c", `web_search="disabled"`)
		if lowEffort && model == "" {
			args = append(args, "-c", `model_reasoning_effort="low"`)
		}
	}
	if model != "" {
		args = append(args, "-m", model)
	}
	return append(args, "-")
}

func (c *Client) runnerFor(ctx context.Context) (*runner, *clicore.APIError) {
	r, err := c.runners.Get(func() string { return ResolveBin(c.cfg.Bin) },
		func(bin string) (*runner, bool, error) { return newRunner(ctx, bin, c.cfg.TempBase) })
	if clicore.IsBinNotFound(err) {
		return nil, codes.MissingError("codex CLI not found on PATH")
	}
	if err != nil {
		return nil, codes.MissingError(err.Error())
	}
	return r, nil
}

// noToolsInstruction keeps the agent answering in text: Punisher only reads
// the final message.
const noToolsInstruction = "Answer directly in your final message. Do not run commands, read or edit files, or browse."

// Create runs one call and returns the final agent message. apiKey is ignored.
func (c *Client) Create(ctx context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	return c.call(ctx, req, nil)
}

// Stream runs one call, passing the final agent message to onDelta when
// the turn completes (see Parser). apiKey is ignored.
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
		return llm.MessagesResult{}, codes.ToLLMError(clicore.NewError(http.StatusBadRequest, clicore.CodeInvalid, "invalid codex model id"))
	}
	r, aerr := c.runnerFor(ctx)
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	if aerr := r.requiredError(); aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	runCtx, done, aerr := c.gate.Admit(ctx, req, codes)
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	defer done()

	stdin := clicore.WithSystem(strings.TrimSpace(req.System+"\n\n"+noToolsInstruction), prompt)
	parser := &Parser{OnDelta: onDelta}
	_, stderr, runErr := r.Run(runCtx, r.Args(model, req.Tier == llm.TierFast), stdin, parser.Feed)
	res, aerr := clicore.Finish(runCtx, ctx, codes, runErr, stderr, parser.Finish)
	if c.fails.Record(aerr, codes) {
		c.status.Reset() // re-read login state on the next status call
	}
	if aerr != nil {
		return llm.MessagesResult{}, codes.ToLLMError(aerr)
	}
	name := model
	if name == "" {
		name = DefaultModelName
	}
	return llm.MessagesResult{Text: res.Text, Model: name, StopReason: "end_turn", LatencyMs: time.Since(start).Milliseconds(),
		Usage: llm.Usage{InputTokens: res.InputTokens, OutputTokens: res.OutputTokens}}, nil
}

// statusTTL is how long Status reuses `--version` / `login status`
// (the UI polls /v1/status).
const statusTTL = 30 * time.Second

// Status reports install/login state: `codex --version` and `codex login
// status` (exit 0 = logged in), cached for statusTTL. It never makes a model call.
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
		out, err := r.Output(ctx, 10*time.Second, "--version")
		probe = clicore.StatusProbe{Version: strings.TrimSpace(out), Broken: err != nil}
		if err == nil {
			probe.LoggedIn = loginStatus(ctx, r)
		}
		if ctx.Err() == nil { // a cut-short probe is not remembered
			c.status.Put(r.Bin, probe)
		}
	}
	if probe.Broken {
		st.Code = CodeCLIMissing
		return st
	}
	st.Version, st.LoggedIn = probe.Version, probe.LoggedIn
	c.fails.Finish(&st, codes)
	return st
}

// loginStatus runs `codex login status`: exit 0 = logged in; a non-zero exit
// saying "not logged in" = logged out; anything else = unknown (nil).
func loginStatus(ctx context.Context, r *runner) *bool {
	stdout, stderr, err := r.Outputs(ctx, 15*time.Second, "login", "status")
	yes, no := true, false
	if err == nil {
		return &yes
	}
	text := strings.ToLower(stdout + " " + stderr)
	if strings.Contains(text, "not logged in") || strings.Contains(text, "logged out") {
		return &no
	}
	return nil
}
