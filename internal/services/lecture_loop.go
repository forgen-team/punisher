package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/forgen-team/punisher/internal/i18n"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/prompts"
)

// Per-attempt time limits of a lecture generation (an attempt = one draft and
// its review). Job limits are at least attempts × (draft + review).
const (
	draftTimeout  = 7 * time.Minute
	reviewTimeout = 3 * time.Minute
	attemptBudget = draftTimeout + reviewTimeout
	// adjudicateTimeout bounds one adjudication (a short call; adjudications
	// of one review run in parallel).
	adjudicateTimeout = 90 * time.Second
)

// lectureLoop is the LLM-backed LoopFuncs of a lecture generation (library,
// regenerate and instant share it): draft with the teacher system prompt
// (outlineJSON may be empty; a rewrite edits the previous draft in place),
// review with the reviewer prompt, and parse = format validation + machine
// checks (Target.parseAndValidate).
func (s *Service) lectureLoop(apiKey string, t Target, outlineJSON string, u *usage) LoopFuncs {
	pc := t.LectureContext()
	vars := prompts.LectureVars(pc)
	lang := t.Plan.Language // 강의와 그 검수·재검·판정은 강의 언어(i18n §1.1)
	teacher := prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockTeacherSystem, vars), lang)
	maxTokens := s.Cfg.LectureMaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	var mu sync.Mutex // adjudications run in parallel and share u
	return LoopFuncs{
		Generate: func(ctx context.Context, rev Revision) (string, error) {
			user := prompts.Lecture(pc, outlineJSON, rev.Notes, t.Prompts)
			if rev.Previous != "" {
				user = prompts.LectureRevision(pc, outlineJSON, rev.Previous, rev.Notes, t.Prompts)
			}
			return s.completeDraft(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel, System: teacher,
				User: user, MaxTokens: maxTokens, Timeout: draftTimeout}, u)
		},
		Review: func(ctx context.Context, markup string) (ReviewOutcome, error) {
			text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
				System: prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockReviewerSystem, vars), lang),
				User:   prompts.Review(pc, markup, t.Prompts), MaxTokens: 4000, Timeout: reviewTimeout}, u)
			if err != nil {
				return ReviewOutcome{}, err
			}
			return ParseReview(text), nil
		},
		Recheck: func(ctx context.Context, in RecheckInput) (RecheckOutcome, error) {
			text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
				System: prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockReviewerSystem, vars), lang),
				User:   prompts.Recheck(pc, in.Findings, in.Changed, t.Prompts), MaxTokens: 3000, Timeout: reviewTimeout}, u)
			if err != nil {
				return RecheckOutcome{}, err
			}
			return ParseRecheck(text), nil
		},
		Adjudicate: func(ctx context.Context, in AdjudicationInput) (AdjudicationResult, error) {
			var lu usage
			text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
				System: prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockAdjudicator, vars), lang),
				User: prompts.Adjudication(pc, prompts.AdjudicationCase{Kind: in.Issue.Kind, Section: in.Issue.Section,
					Quote: in.Issue.Quote, Message: in.Issue.Message, ReaderImpact: in.Issue.ReaderImpact, Blocks: in.Blocks}),
				MaxTokens: 800, Timeout: adjudicateTimeout, Priority: llm.ShortUnlessBackground(ctx)}, &lu)
			mu.Lock()
			u.input, u.output = u.input+lu.input, u.output+lu.output
			if lu.model != "" {
				u.model = lu.model
			}
			mu.Unlock()
			if err != nil {
				return AdjudicationResult{}, err
			}
			return ParseAdjudication(text), nil
		},
		Parse:             t.parseAndValidate,
		DraftTimeout:      draftTimeout,
		ReviewTimeout:     reviewTimeout,
		AdjudicateTimeout: adjudicateTimeout,
	}
}

// completeDraft is complete for a lecture draft: output cut at max_tokens is
// returned as is, so it fails to parse (lecture.ErrIncomplete) and counts as
// one rewrite instead of failing the job (quality design v2.2 §2.3).
func (s *Service) completeDraft(ctx context.Context, apiKey string, req llm.MessagesRequest, u *usage) (string, error) {
	res, err := s.LLM.Create(ctx, apiKey, req)
	u.add(res)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(res.Text) == "" {
		return "", errors.New("model returned empty output")
	}
	return res.Text, nil
}

// maxUserReasons bounds the failure summary shown to the learner.
const maxUserReasons = 3

// Learner-facing reason texts (stored Korean; codes in internal/i18n, which the web translates).
var (
	reasonFormat    = i18n.ReasonKo(i18n.ReasonFormat)
	reasonTruncated = i18n.ReasonKo(i18n.ReasonTruncated)
	reasonVolume    = i18n.ReasonKo(i18n.ReasonVolume)
	reasonBlocks    = i18n.ReasonKo(i18n.ReasonBlocks)
	reasonLeak      = i18n.ReasonKo(i18n.ReasonLeak)
	reasonTimeout   = i18n.ReasonKo(i18n.ReasonTimeout)
	reasonReview    = i18n.ReasonKo(i18n.ReasonReview)
	// ReasonGenerationError replaces raw LLM / storage errors in the stream.
	ReasonGenerationError = i18n.ReasonKo(i18n.ReasonGenerationError)
)

// userReasons summarizes why an attempt did not pass, in short Korean lines
// safe to show (raw errors and quotes are never included), most important
// first, at most maxUserReasons. Only blocking findings are reasons (advisory
// ones never stop publishing).
func userReasons(a Attempt) []string {
	var out []string
	add := func(r string) {
		for _, x := range out {
			if x == r {
				return
			}
		}
		out = append(out, r)
	}
	switch {
	case a.Err != nil && isAttemptTimeout(a.Err):
		add(reasonTimeout)
	case a.Err != nil:
		add(ReasonGenerationError)
	case errors.Is(a.ParseErr, lecture.ErrIncomplete):
		add(reasonTruncated)
	case a.ParseErr != nil:
		add(reasonFormat)
	}
	for _, sev := range []string{"critical", "major", "minor", ""} {
		for _, i := range a.Review.Issues {
			if i.Severity == sev && i.IsBlocking() {
				add(i18n.ReasonKo(i18n.ReviewReasonCode(i.Kind)))
			}
		}
	}
	for _, i := range a.blockingIssues() {
		add(issueReason(i))
	}
	if len(out) == 0 && a.Reviewed && a.Review.Blocking() {
		add(reasonReview)
	}
	if len(out) > maxUserReasons {
		out = out[:maxUserReasons]
	}
	return out
}

// issueReason maps a validation / machine-check issue to its summary.
func issueReason(i lecture.Issue) string {
	m := i.Message
	switch {
	case strings.Contains(m, "prose must be"):
		return reasonVolume
	case strings.Contains(m, "own concept") || strings.Contains(m, "meta phrase") || strings.Contains(m, "section id"):
		return reasonLeak
	case strings.Contains(m, "@@keypoint") || strings.Contains(m, "@@trace") || strings.Contains(m, "@@exercise") ||
		strings.Contains(m, "@@pause") || strings.Contains(m, "@@summary") || strings.Contains(m, "@@answer"):
		return reasonBlocks
	}
	return reasonFormat
}
