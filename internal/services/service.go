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
	model  string
	input  int
	output int
}

func (u *usage) add(r llm.MessagesResult) {
	u.input += r.Usage.InputTokens
	u.output += r.Usage.OutputTokens
	if r.Model != "" {
		u.model = r.Model
	}
}

func (u usage) outcome(status, reason string, err error, lectureID string) store.JobOutcome {
	o := store.JobOutcome{Status: status, Reason: reason, LectureID: lectureID, Model: u.model,
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

// Stable CLI failure codes (see internal/claudecli) used for user messages.
const (
	codeCLIMissing  = "claude_unavailable"
	codeNotLoggedIn = "claude_not_logged_in"
	codeRateLimited = "claude_rate_limited"
	codeQueueFull   = "queue_full"
	codeDailyCap    = "daily_cap_reached"
	codeTimeout     = "claude_timeout"
)

// Korean fix-it texts shown when the local claude CLI cannot generate.
const (
	MsgClaudeMissing   = "Claude Code(claude 명령)를 찾을 수 없습니다. Claude Code를 설치한 뒤 터미널에서 `claude`를 한 번 실행해 로그인하고 다시 시도해 주세요."
	MsgClaudeLoggedOut = "Claude Code에 로그인되어 있지 않습니다. 터미널에서 `claude`를 실행해 로그인(/login)한 뒤 다시 시도해 주세요."
)

// checkErrorText is the stored error of a failed check step (shown on the
// check screen): the Korean local-CLI guidance for generation failures.
func checkErrorText(err error) string {
	if msg := DescribeLLMError(err); msg != "" {
		return msg
	}
	return err.Error()
}

// DescribeLLMError returns a Korean, user-facing description of a generation
// failure ("" when err is not a generation error).
func DescribeLLMError(err error) string {
	var le *llm.LLMError
	code := ""
	if errors.As(err, &le) {
		code = le.Code
	}
	switch {
	case code == codeCLIMissing:
		return MsgClaudeMissing
	case code == codeNotLoggedIn:
		return MsgClaudeLoggedOut
	case code == codeRateLimited:
		return "Claude 사용 한도에 도달했습니다. 한도가 풀린 뒤(보통 몇 시간 이내) 다시 시도해 주세요."
	case code == codeQueueFull:
		return "다른 생성 작업이 많이 밀려 있습니다. 잠시 후 다시 시도해 주세요."
	case code == codeDailyCap:
		return "오늘 설정한 Claude 호출 한도를 모두 사용했습니다. 내일 다시 시도해 주세요."
	case code == codeTimeout || errors.Is(err, llm.ErrTimeout):
		return "Claude 응답이 너무 오래 걸려 중단했습니다. 다시 시도해 주세요."
	case errors.Is(err, llm.ErrGenerationUnavailable):
		return MsgClaudeLoggedOut
	case errors.Is(err, ErrUnusableOutput):
		return "AI가 만든 강의 형식이 올바르지 않아 저장하지 못했습니다. 다시 시도해 주세요."
	case le != nil:
		return "Claude 호출에 실패했습니다: " + le.Message
	}
	return ""
}
