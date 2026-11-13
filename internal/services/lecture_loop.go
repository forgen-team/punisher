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

// lectureLoop is the LLM-backed LoopFuncs of a lecture generation (library,
// regenerate and instant share it): draft with the teacher system prompt
// (outlineJSON may be empty), review with the reviewer prompt, and
// parse = format validation + machine checks (Target.parseAndValidate).
func (s *Service) lectureLoop(apiKey string, t Target, outlineJSON string, u *usage) LoopFuncs {
	pc := t.LectureContext()
	vars := prompts.LectureVars(pc)
	teacher := t.Prompts.Text(prompts.BlockTeacherSystem, vars)
	maxTokens := s.Cfg.LectureMaxTokens
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}
	return LoopFuncs{
		Generate: func(ctx context.Context, notes string) (string, error) {
			return s.completeDraft(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel, System: teacher,
				User: prompts.Lecture(pc, outlineJSON, notes, t.Prompts), MaxTokens: maxTokens, Timeout: 6 * time.Minute}, u)
		},
		Review: func(ctx context.Context, markup string) (ReviewOutcome, error) {
			text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
				System: t.Prompts.Text(prompts.BlockReviewerSystem, vars),
				User:   prompts.Review(pc, markup, t.Prompts), MaxTokens: 4000, Timeout: 4 * time.Minute}, u)
			if err != nil {
				return ReviewOutcome{}, err
			}
			return ParseReview(text), nil
		},
		Parse: t.parseAndValidate,
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
	// ReasonGenerationError replaces raw LLM / storage errors in the stream.
	ReasonGenerationError = "강의를 만드는 중 오류가 났어요. 잠시 후 다시 만들어 주세요"
)

// userReasons summarizes why an attempt did not pass, in short Korean lines
// safe to show (raw errors and quotes are never included), most important
// first, at most maxUserReasons.
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
	case errors.Is(a.ParseErr, lecture.ErrIncomplete):
		add(reasonTruncated)
	case a.ParseErr != nil:
		add(reasonFormat)
	}
	for _, sev := range []string{"critical", "major"} {
		for _, i := range a.Review.Issues {
			if i.Severity == sev {
				if r, ok := reviewReasons[i.Kind]; ok {
					add(r)
				} else {
					add(reasonReview)
				}
			}
		}
	}
	for _, i := range a.Issues {
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
