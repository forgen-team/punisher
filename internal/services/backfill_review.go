package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/lecture"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// BackfillReviewsPerTick bounds the stored lectures reviewed per scheduler
// tick (quality design v2.2 §2.2 — it also runs on the production DB).
const BackfillReviewsPerTick = 2

// reasonUnreviewable flags a lecture whose concept is gone (it cannot be reviewed).
const reasonUnreviewable = "강의의 개념을 찾을 수 없어 검수하지 못했어요"

// ReviewerBackfill marks lecture_reviews rows written by the backfill review.
const ReviewerBackfill = "backfill"

// BackfillReport summarises one backfill run.
type BackfillReport struct {
	Passed  []string `json:"passed"`
	Flagged []string `json:"flagged"`
	Failed  []string `json:"failed"`
}

// BackfillReviews reviews up to BackfillReviewsPerTick published lectures
// without a pass review (requested ones first): a pass is recorded and clears
// the flag; a fail flags the lecture with the reasons (the "검수에서 문제가
// 발견된 강의" banner). Off with Cfg.BackfillReviewEnabled=false or without an owner.
func (s *Service) BackfillReviews(ctx context.Context) (BackfillReport, error) {
	var rep BackfillReport
	owner := s.Cfg.OwnerUserID
	if owner == "" || !s.Cfg.BackfillReviewEnabled {
		return rep, nil
	}
	list, err := s.Store.LecturesToReview(ctx, BackfillReviewsPerTick)
	if err != nil || len(list) == 0 {
		return rep, err
	}
	if err := s.CheckBudget(ctx, owner, false); err != nil {
		return rep, err
	}
	apiKey, err := s.Keys.UserKey(ctx, owner)
	if err != nil {
		return rep, err
	}
	for _, lec := range list {
		passed, err := s.backfillOne(ctx, apiKey, owner, lec)
		switch {
		case err != nil:
			logf("backfill review %s: %v", lec.ID, err)
			rep.Failed = append(rep.Failed, lec.ID)
			if errors.Is(err, llm.ErrGenerationUnavailable) || errors.Is(err, llm.ErrRateLimited) {
				return rep, err
			}
		case passed:
			rep.Passed = append(rep.Passed, lec.ID)
		default:
			rep.Flagged = append(rep.Flagged, lec.ID)
		}
	}
	return rep, nil
}

// backfillOne runs the review pass on a stored lecture (rendered back to
// markup; lectures without material blocks get the legacy note) and records
// the outcome. An LLM error records nothing (the lecture stays in the queue).
func (s *Service) backfillOne(ctx context.Context, apiKey, owner string, lec models.Lecture) (bool, error) {
	var c lecture.Content
	if err := json.Unmarshal(lec.Content, &c); err != nil {
		return false, fmt.Errorf("decode content: %w", err)
	}
	userID := lec.RequestedBy
	if userID == "" {
		userID = owner
	}
	t, err := s.TargetForLecture(ctx, userID, lec)
	if errors.Is(err, store.ErrNotFound) { // 개념이 지도에서 빠진 강의: 대기열을 막지 않도록 기록하고 표시한다.
		return false, s.recordBackfill(ctx, lec.ID, ReviewOutcome{Verdict: "fail", Summary: reasonUnreviewable}, "", reasonUnreviewable)
	}
	if err != nil {
		return false, err
	}
	set := prompts.Defaults()
	if lec.Kind == learning.JobInstant {
		set, t.Owner = s.UserPrompts(ctx, userID), userID
	}
	pc := t.LectureContext()
	build := prompts.Review
	if !lecture.HasMaterialBlocks(c) {
		build = prompts.ReviewLegacy
	}
	var u usage
	text, err := s.complete(ctx, apiKey, llm.MessagesRequest{Model: s.Cfg.LectureModel,
		System: set.Text(prompts.BlockReviewerSystem, prompts.LectureVars(pc)),
		User:   build(pc, lecture.RenderMarkup(c), set), MaxTokens: 4000, Timeout: reviewTimeout}, &u)
	s.recordUsage(context.WithoutCancel(ctx), owner, "lecture_review", lec.ConceptID, u, err)
	if err != nil {
		return false, err
	}
	outcome := ParseReview(text)
	reason := strings.Join(userReasons(Attempt{Review: outcome, Reviewed: true}), "; ")
	return !outcome.Blocking(), s.recordBackfill(ctx, lec.ID, outcome, u.model, reason)
}

// recordBackfill stores the backfill review row (reviewer backfill — the
// lecture leaves the queue) and its result on the lecture.
func (s *Service) recordBackfill(ctx context.Context, lectureID string, outcome ReviewOutcome, model, reason string) error {
	passed := !outcome.Blocking()
	verdict := "fail"
	if passed {
		verdict = "pass"
	}
	if err := s.Store.InsertReview(ctx, lectureID, models.LectureReview{Verdict: verdict, Summary: outcome.Summary,
		Issues: issuesJSON(outcome.Issues), ValidationIssues: issuesJSON(nil), ReaderGaps: issuesJSON(outcome.ReaderGaps),
		Attempt: 1, Model: model, Reviewer: ReviewerBackfill}); err != nil {
		return err
	}
	return s.Store.SetLectureReviewOutcome(ctx, lectureID, passed, reason)
}
