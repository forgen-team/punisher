package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/llm"
)

// Per-attempt time limits of a lecture generation (an attempt = one draft and
// its review). Job limits are at least attempts × (draft + review).
const (
	draftTimeout  = 7 * time.Minute
	reviewTimeout = 3 * time.Minute
	attemptBudget = draftTimeout + reviewTimeout
)

// lectureLoop is the LLM-backed LoopFuncs of a lecture generation (library,
// regenerate and instant share it): draft with the teacher system prompt
// (outlineJSON may be empty; a rewrite edits the previous draft in place),
// review with the reviewer prompt, and parse = format validation + machine
// checks (Target.parseAndValidate).
func (s *Service) lectureLoop(apiKey string, t Target, outlineJSON string, u *usage) LoopFuncs {
	pc := t.LectureContext()
	vars := prompts.LectureVars(pc)
	teacher := t.Prompts.Text(prompts.BlockTeacherSystem, vars)
	maxTokens := s.Cfg.LectureMaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
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
				System: t.Prompts.Text(prompts.BlockReviewerSystem, vars),
				User:   prompts.Review(pc, markup, t.Prompts), MaxTokens: 4000, Timeout: reviewTimeout}, u)
			if err != nil {
				return ReviewOutcome{}, err
			}
			return ParseReview(text), nil
		},
		Parse:         t.parseAndValidate,
		DraftTimeout:  draftTimeout,
		ReviewTimeout: reviewTimeout,
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

// reviewReasons are the learner-facing summaries of review findings by kind.
var reviewReasons = map[string]string{
	"factual_error":     "사실과 다르거나 확인이 필요한 설명이 있었어요",
	"structure":         "핵심 정의나 논지 등 강의 구성에 문제가 있었어요",
	"pedagogy":          "손계산·예측·연습 같은 학습 활동이 부족했어요",
	"omission":          "학습 목표 일부를 충분히 다루지 못했어요",
	"assumed_knowledge": "설명 없이 쓰인 용어가 있었어요",
	"overclaim":         "근거 없이 단정한 문장이 있었어요",
	"source":            "근거 문서가 공식 문서가 아니었어요",
	"terminology":       "용어 정의가 일관되지 않았어요",
	"typo":              "실행되지 않는 코드나 오타가 있었어요",
}

const (
	reasonFormat    = "강의 형식을 지키지 못했어요"
	reasonTruncated = "강의가 길어져 끝까지 쓰지 못했어요"
	reasonVolume    = "분량이 기준에 맞지 않았어요"
	reasonBlocks    = "정의·손계산·예측·연습·요약 구성이 기준에 못 미쳤어요"
	reasonLeak      = "생성 과정의 문구나 잘못된 링크가 남아 있었어요"
	reasonReview    = "검수 기준을 통과하지 못했어요"
	reasonTimeout   = "강의를 쓰는 데 시간이 너무 오래 걸렸어요"
	// ReasonGenerationError replaces raw LLM / storage errors in the stream.
	ReasonGenerationError = "강의를 만드는 중 오류가 났어요. 잠시 후 다시 만들어 주세요"
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
				if r, ok := reviewReasons[i.Kind]; ok {
					add(r)
				} else {
					add(reasonReview)
				}
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
