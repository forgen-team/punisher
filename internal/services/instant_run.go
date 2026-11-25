package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/llmjson"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
)

const (
	// InstantRewrites: a draft that fails the checks or the review is rewritten
	// with the findings at most this many times (quality design v2.2 §2.1).
	InstantRewrites = 2
	// outlineLiteTimeout bounds the short plan of an instant lecture.
	outlineLiteTimeout = 90 * time.Second
	// instantTimeout bounds the whole pipeline: the short plan and every
	// attempt with its full draft + review limit. It stays under the
	// scheduler's stale-job age so a live job is never swept.
	instantTimeout = outlineLiteTimeout + (InstantRewrites+1)*attemptBudget
)

// LongestJobTimeout is the longest generation job limit; the scheduler's
// stale-job sweep must wait longer than this.
const LongestJobTimeout = max(instantTimeout, libraryJobTimeout)

// Job reasons (generation_jobs.reason) of a failed instant generation.
const (
	JobReasonReviewFailed = "review_failed" // error = learner-facing reasons, one per line
	JobReasonLLMError     = "llm_error"
	JobReasonStoreError   = "store_error"
)

// runInstant generates one instant lecture through the reviewed pipeline,
// detached from the HTTP request: draft (not streamed) → format validation →
// machine checks → review; a failing draft is rewritten with the findings up
// to InstantRewrites times. Only a passing draft is saved and published; when
// none passes nothing is published and the stream ends with stage failed and
// short reasons. Stage events report progress (see StageEvent).
func (s *Service) runInstant(apiKey, userID, jobID string, t Target, emit func(Event)) {
	ctx, cancel := context.WithTimeout(context.Background(), instantTimeout)
	defer cancel()
	attempt := 0
	var u usage
	var result LoopResult
	fail := func(code string, err error, reasons []string) {
		logf("instant job %s failed (%s): %v", jobID, code, err)
		o := u.outcome("failed", code, err, "")
		// job.error 는 화면에 그대로 보이므로 요약만 남긴다(원문 오류는 로그와 attempt_log 에).
		o.Error = strings.Join(reasons, "\n")
		if len(result.Attempts) > 0 {
			o.Attempts = result.AttemptLog()
		}
		if ferr := s.Store.FinishJob(context.WithoutCancel(ctx), jobID, o); ferr != nil {
			logf("finish job %s: %v", jobID, ferr)
		}
		emit(stageEvent(StageFailed, attempt, InstantRewrites, "", reasons))
		emit(Event{Type: EventError, Data: map[string]string{"code": code, "message": strings.Join(reasons, " · ")}})
	}
	if err := s.Store.StartJob(ctx, jobID); err != nil {
		fail(JobReasonStoreError, err, []string{ReasonGenerationError})
		return
	}

	outline := ""
	if s.Cfg.InstantOutline {
		emit(stageEvent(StagePlanning, 0, InstantRewrites, "", nil))
		outline = s.instantOutline(ctx, apiKey, t, &u)
	}
	loop := s.lectureLoop(apiKey, t, outline, &u)
	loop.SkipReviewOnIssues = true
	loop.NoAdvisoryRewrite = true // advisory only → publish at once (shorter wait)
	loop.OnStage = func(stage string, n int) {
		attempt = n
		emit(stageEvent(stage, n, InstantRewrites, "", nil))
	}
	result, err := RunReviewLoop(ctx, InstantRewrites+1, loop)
	switch {
	case errors.Is(err, ErrUnusableOutput) && len(result.Attempts) > 0: // 어떤 초안도 형식을 지키지 못함
		fail(JobReasonReviewFailed, err, userReasons(result.Attempts[len(result.Attempts)-1]))
		return
	case err != nil:
		fail(JobReasonLLMError, err, []string{llmFailReason(err)})
		return
	case result.Status != "published":
		fail(JobReasonReviewFailed, nil, userReasons(result.Attempts[len(result.Attempts)-1]))
		return
	}
	saved, err := s.storeAttempts(ctx, t, jobID, result, u.model, u.provider)
	if err != nil {
		fail(JobReasonStoreError, err, []string{ReasonGenerationError})
		return
	}
	done := u.outcome("succeeded", saved.Status, nil, saved.ID)
	done.Attempts = result.AttemptLog()
	if err := s.Store.FinishJob(ctx, jobID, done); err != nil {
		logf("finish job %s: %v", jobID, err)
	}
	if err := s.Store.ResolveLinkJob(ctx, jobID, saved.ID); err != nil {
		logf("resolve links of job %s: %v", jobID, err)
	}
	emit(stageEvent(StagePublished, attempt, InstantRewrites, saved.ID, nil))
	emit(Event{Type: EventDone, Data: map[string]any{
		"lectureId": saved.ID, "status": saved.Status, "flagged": false, "issues": []any{}, "attempts": len(result.Attempts),
	}})
}

// instantOutline is the short plan of an instant lecture (thesis check, key
// definitions, misconceptions, scene numbers — research #13) as compact JSON,
// "" when the call fails or its output is not one JSON object (the draft is
// then written without it, as before).
func (s *Service) instantOutline(ctx context.Context, apiKey string, t Target, u *usage) string {
	pc := t.LectureContext()
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
		System: prompts.WithOutputLanguage(t.Prompts.Text(prompts.BlockTeacherSystem, prompts.LectureVars(pc)), t.Plan.Language),
		User:   prompts.OutlineLite(pc), MaxTokens: 1500, Timeout: outlineLiteTimeout}, u)
	if err != nil {
		logf("instant outline of %s: %v (drafting without it)", t.Plan.ConceptID, err)
		return ""
	}
	plan, err := llmjson.TryParseJSON[map[string]json.RawMessage](text)
	if err != nil || len(plan) == 0 {
		logf("instant outline of %s unreadable (drafting without it)", t.Plan.ConceptID)
		return ""
	}
	raw, err := json.Marshal(plan)
	if err != nil {
		return ""
	}
	return string(raw)
}

// llmFailReason is the learner-facing reason of a failed generation call:
// 단독판은 로컬 Claude Code CLI 의 미설치·미로그인·한도를 한국어로 알린다
// (DescribeLLMError, docs/SYNC.md), 그 밖은 witems 와 같은 일반 문구.
func llmFailReason(err error) string {
	if msg := DescribeLLMError(err); msg != "" {
		return msg
	}
	return ReasonGenerationError
}

// instantFailReasons: job reasons whose error column holds learner-facing reasons.
var instantFailReasons = map[string]bool{JobReasonReviewFailed: true, JobReasonLLMError: true, JobReasonStoreError: true}

// InstantFinalEvents are the events for a job no longer in the hub (finished
// or another process), from its generation_jobs row: the terminal stage
// followed by the legacy done/error.
func InstantFinalEvents(job models.GenerationJob) []Event {
	switch job.Status {
	case "succeeded":
		return []Event{stageEvent(StagePublished, 0, InstantRewrites, job.LectureID, nil),
			{Type: EventDone, Data: map[string]any{"lectureId": job.LectureID, "status": job.Reason}}}
	case "failed", "skipped":
		reasons := []string{ReasonGenerationError}
		if instantFailReasons[job.Reason] && strings.TrimSpace(job.Error) != "" {
			reasons = strings.Split(job.Error, "\n")
		}
		return []Event{stageEvent(StageFailed, 0, InstantRewrites, "", reasons),
			{Type: EventError, Data: map[string]string{"code": job.Status, "message": strings.Join(reasons, " · ")}}}
	}
	return []Event{stageEvent(StageWriting, 0, InstantRewrites, "", nil),
		{Type: EventStatus, Data: map[string]string{"stage": job.Status}}}
}
