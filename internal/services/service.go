// Package services implements Punisher's LLM-backed workflows: library
// lecture pipeline, instant streaming lectures, concept resolution, glosses
// and contextual Q&A. Pure decision logic lives in internal/learning and
// internal/lecture; this package wires it to the store and the LLM.
package services

import (
	"context"
	"errors"
	"fmt"
	"github.com/forgen-team/punisher/internal/glossary"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/i18n"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/store"
)

var (
	// ErrNoAPIKey: the user has no Anthropic BYOK key registered.
	ErrNoAPIKey = errors.New("anthropic API key is not registered")
	// ErrBudgetExceeded: the daily generation budget is used up.
	ErrBudgetExceeded = errors.New("daily generation budget exceeded")
	// ErrNotOwner: LLM features are limited to PUNISHER_OWNER_USER_ID.
	ErrNotOwner = errors.New("LLM features are limited to the owner account")
	// ErrUnusableOutput: the model output could not be parsed into a lecture.
	ErrUnusableOutput = errors.New("model output is not a usable lecture")
)

// LLM is the generation client (claudecli.Client; faked in tests).
type LLM interface {
	Create(ctx context.Context, apiKey string, req llm.MessagesRequest) (llm.MessagesResult, error)
	Stream(ctx context.Context, apiKey string, req llm.MessagesRequest, onDelta func(string) error) (llm.MessagesResult, error)
}

// KeySource resolves per-user API keys (unused by the local CLI; kept so the
// services stay provider-agnostic).
type KeySource interface {
	// RequestKey uses the caller's session (user request) and falls back to the user id.
	RequestKey(ctx context.Context, userID string) (string, error)
	// UserKey is for background jobs (no session).
	UserKey(ctx context.Context, userID string) (string, error)
}

// GatewayKeys is the KeySource for the local claude CLI: generation uses the
// logged-in Claude Code account, so no per-user key exists. The returned
// placeholder is ignored by claudecli.Client.
type GatewayKeys struct{}

const gatewayKeyPlaceholder = "claude-cli"

func (GatewayKeys) RequestKey(context.Context, string) (string, error) {
	return gatewayKeyPlaceholder, nil
}
func (GatewayKeys) UserKey(context.Context, string) (string, error) {
	return gatewayKeyPlaceholder, nil
}

// Config holds generation settings.
type Config struct {
	LectureModel      string // lectures, review, Q&A
	FastModel         string // term glosses, concept resolution
	OwnerUserID       string // "" = no owner restriction, background jobs disabled
	DailyLectureLimit int    // lecture generations per KST day (library + instant)
	DailyTokenLimit   int    // tokens per KST day across all LLM calls
	LibraryAhead      int    // library lectures to keep ready (0 = no ahead pre-generation)
	// Night window (KST hours [NightStartHour, NightEndHour)): keep NightAhead
	// recommended lectures ready while the owner sleeps (0 or start == end = off).
	NightAhead       int
	NightStartHour   int
	NightEndHour     int
	StudyPregenerate bool // pre-generate the starting lecture of newly resolved study requests
	LectureMaxTokens int
	// LibraryUsesUserPrompts applies the owner's prompt overrides to library
	// lectures too (standalone: the single user is the library's only reader).
	LibraryUsesUserPrompts bool
	// Mastery checks (design §4.5).
	CheckModel           string // grading ("" = LectureModel)
	CheckRegradeModel    string // dispute regrade ("" = FastModel)
	DailyCheckTokenLimit int    // check tokens per KST day (0 = unlimited)
	// BackfillReviewEnabled: review stored lectures without a pass review, BackfillReviewsPerTick per scheduler tick.
	BackfillReviewEnabled bool
}

// Service bundles dependencies.
type Service struct {
	Store store.Store
	LLM   LLM
	Keys  KeySource
	Cfg   Config
	Hub   *Hub
	Now   func() time.Time
	// Signals: RSS interests/trends for next-chapter suggestions (nil = off, e.g. standalone).
	Signals SignalSource
	// Glossary: the shipped term dictionary (nil = none — glosses are generated as before).
	Glossary *glossary.Glossary
}

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// AllowUser reports whether userID may use LLM features.
func (s *Service) AllowUser(userID string) bool {
	return s.Cfg.OwnerUserID == "" || s.Cfg.OwnerUserID == userID
}

// CheckBudget enforces the daily (KST) budget. lecture=true also checks the
// lecture-count limit.
func (s *Service) CheckBudget(ctx context.Context, userID string, lecture bool) error {
	count, tokens, err := s.Store.UsageSince(ctx, userID, learning.KSTDayStart(s.now()))
	if err != nil {
		return fmt.Errorf("load usage: %w", err)
	}
	if s.Cfg.DailyTokenLimit > 0 && tokens >= s.Cfg.DailyTokenLimit {
		return fmt.Errorf("%w: %d/%d tokens", ErrBudgetExceeded, tokens, s.Cfg.DailyTokenLimit)
	}
	if lecture && s.Cfg.DailyLectureLimit > 0 && count >= s.Cfg.DailyLectureLimit {
		return fmt.Errorf("%w: %d/%d lectures", ErrBudgetExceeded, count, s.Cfg.DailyLectureLimit)
	}
	return nil
}

// usage accumulates token usage across calls of one job.
type usage struct {
	model    string
	provider string // the AI that ran the calls (Router sets MessagesResult.Provider)
	input    int
	output   int
}

func (u *usage) add(r llm.MessagesResult) {
	u.input += r.Usage.InputTokens
	u.output += r.Usage.OutputTokens
	if r.Model != "" {
		u.model = r.Model
	}
	if r.Provider != "" {
		u.provider = r.Provider
	}
}

func (u usage) outcome(status, reason string, err error, lectureID string) store.JobOutcome {
	o := store.JobOutcome{Status: status, Reason: reason, LectureID: lectureID, Model: u.model, Provider: u.provider,
		InputTokens: u.input, OutputTokens: u.output}
	if err != nil {
		o.Error = err.Error()
	}
	return o
}

// recordUsage stores an accounting-only job and logs (does not fail) on error.
func (s *Service) recordUsage(ctx context.Context, userID, kind, conceptID string, u usage, err error) {
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	if rerr := s.Store.RecordUsage(ctx, userID, kind, conceptID, u.outcome(status, "", err, "")); rerr != nil {
		logf("record %s usage: %v", kind, rerr)
	}
}

// complete runs one non-streaming call and rejects truncated/empty output.
func (s *Service) complete(ctx context.Context, apiKey string, req llm.MessagesRequest, u *usage) (string, error) {
	res, err := s.LLM.Create(ctx, apiKey, req)
	u.add(res)
	if err != nil {
		return "", err
	}
	if res.Truncated() {
		return "", fmt.Errorf("model output truncated at max_tokens (%d)", req.MaxTokens)
	}
	if strings.TrimSpace(res.Text) == "" {
		return "", errors.New("model returned empty output")
	}
	return res.Text, nil
}

// Stable CLI failure codes (see internal/clicore): the claude ones; codex
// and agy use the same suffixes (codex_unavailable, agy_not_logged_in, …).
const (
	codeCLIMissing  = "claude_unavailable"
	codeNotLoggedIn = "claude_not_logged_in"
	codeRateLimited = "claude_rate_limited"
	codeQueueFull   = "queue_full"
	codeDailyCap    = "daily_cap_reached"
	codeTimeout     = "claude_timeout"
)

// Korean fix-it texts shown when the local claude CLI cannot generate (the
// stored text of their i18n reason codes, which the web translates).
var (
	MsgClaudeMissing   = i18n.ReasonKo(i18n.ReasonClaudeMissing)
	MsgClaudeLoggedOut = i18n.ReasonKo(i18n.ReasonClaudeLoggedOut)
)

// ProviderMessage is the stored (Korean) text of a failure kind (missing,
// logged_out, rate_limited, …) of provider (claude | codex | agy).
func ProviderMessage(provider, kind string) string {
	return i18n.ReasonKo(i18n.ProviderReason(provider, kind))
}

// checkErrorText is the stored error of a failed check step (shown on the
// check screen): the Korean local-CLI guidance for generation failures.
func checkErrorText(err error) string {
	if msg := DescribeLLMError(err); msg != "" {
		return msg
	}
	return err.Error()
}

// DescribeLLMError returns the stored (Korean) text of a generation failure
// ("" when err is not a generation error); LLMErrorReason gives its code.
// The text names the AI that failed ("Codex(codex 명령)를 찾을 수 없습니다").
func DescribeLLMError(err error) string {
	if code := LLMErrorReason(err); code != "" {
		return i18n.ReasonKo(code)
	}
	return ""
}

// failureKind is the reason kind of a CLI code of provider ("" = none):
// <provider>_unavailable → missing, _not_logged_in → logged_out, …
func failureKind(provider, code string) string {
	switch code {
	case "":
		return ""
	case codeQueueFull:
		return "queue_full"
	case codeDailyCap:
		return "daily_cap"
	case provider + "_unavailable":
		return "missing"
	case provider + "_not_logged_in":
		return "logged_out"
	case provider + "_rate_limited":
		return "rate_limited"
	case provider + "_timeout":
		return "timeout"
	}
	return ""
}

// LLMErrorReason is the i18n reason code of a generation failure ("" when err
// is not a generation error), for the provider that failed (LLMError.Provider;
// "" = claude, as from claude-gateway). The CLI's own message of an
// unclassified failure is logged, not shown.
func LLMErrorReason(err error) string {
	var le *llm.LLMError
	code, provider := "", llm.ProviderClaude
	if errors.As(err, &le) {
		code = le.Code
		if le.Provider != "" {
			provider = le.Provider
		}
	}
	reason := func(kind string) string { return i18n.ProviderReason(provider, kind) }
	if kind := failureKind(provider, code); kind != "" {
		return reason(kind)
	}
	switch {
	case errors.Is(err, llm.ErrTimeout):
		return reason("timeout")
	case errors.Is(err, llm.ErrGenerationUnavailable):
		return reason("logged_out")
	case errors.Is(err, ErrUnusableOutput):
		return reason("bad_output")
	case le != nil:
		logf("%s call failed: %s", provider, le.Message)
		return reason("failed")
	}
	return ""
}
