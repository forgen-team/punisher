package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// RegenerateLecture generates a new version of lecture lectureID (same concept,
// objectives, depth) with the user's current prompt blocks through the full
// outline → lecture → review loop, owned by the user. It bypasses the
// published-lecture reuse on purpose (the user asked for a new version) and
// links the result under the original (lecture_links kind=regenerated).
// Progress is streamed as stage events (and the legacy done/error) on the instant stream URL.
func (s *Service) RegenerateLecture(ctx context.Context, userID, lectureID string) (InstantResponse, error) {
	lec, err := s.Store.GetLecture(ctx, userID, lectureID)
	if err != nil {
		return InstantResponse{}, err
	}
	// 다시 만들기는 사용자 언어의 새 버전(i18n §1.2).
	t, err := s.TargetForLecture(ctx, userID, lec, s.Store.LocaleFor(ctx, userID))
	if err != nil {
		return InstantResponse{}, err
	}
	t.Prompts, t.PromptBlocks, t.Owner = s.UserPrompts(ctx, userID), prompts.FullBlocks, userID
	t.personal.signals = s.signalLines(ctx, userID, lec)
	resp := InstantResponse{ConceptID: lec.ConceptID}

	target := learning.TargetKey(t.Plan)
	if jobID, ok := s.Hub.ActiveJob(target); ok {
		resp.Status, resp.JobID, resp.StreamURL = InstantAttached, jobID, streamURL(jobID)
		return resp, nil
	}
	if err := s.CheckBudget(ctx, userID, true); err != nil {
		return InstantResponse{}, err
	}
	apiKey, err := s.Keys.RequestKey(ctx, userID)
	if err != nil {
		return InstantResponse{}, err
	}
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: fmt.Sprintf("%s:regen:%s:%s", learning.JobInstant, target, store.RandomToken()),
		Kind:           learning.JobInstant, UserID: userID, ConceptID: lec.ConceptID,
		ObjectiveKey: t.Plan.ObjectiveKey, Depth: t.Plan.Depth, Reason: "regenerate", Language: t.Plan.Language,
	})
	if errors.Is(err, store.ErrJobActive) {
		return InstantResponse{}, ErrTargetBusy
	}
	if err != nil {
		return InstantResponse{}, err
	}
	if err := s.Store.LinkDeeper(ctx, store.DeeperLink{
		UserID: userID, ParentLecture: lec.ID, ChildJobID: job.ID, ChildConceptID: lec.ConceptID,
		Kind: store.LinkRegeneratedKind,
	}); err != nil {
		logf("link regenerated %s -> job %s: %v", lec.ID, job.ID, err)
	}

	jobID, started := s.Hub.Start(job.ID, target, func(emit func(Event)) {
		s.runRegenerate(apiKey, job.ID, t, emit)
	})
	resp.JobID, resp.StreamURL, resp.Status = jobID, streamURL(jobID), InstantStarted
	if !started {
		resp.Status = InstantAttached
		if ferr := s.Store.FinishJob(ctx, job.ID, store.JobOutcome{Status: "skipped", Reason: "attached to " + jobID}); ferr != nil {
			logf("skip duplicate job %s: %v", job.ID, ferr)
		}
	}
	return resp, nil
}

// runRegenerate runs the reviewed pipeline detached from the request, streaming
// stage events (maxAttempts = its one regeneration) and the legacy done/error.
func (s *Service) runRegenerate(apiKey, jobID string, t Target, emit func(Event)) {
	ctx := context.Background()
	const rewrites = libraryAttempts - 1
	lec, err := s.runLibraryJob(ctx, apiKey, jobID, t, func(stage string, attempt int) {
		emit(stageEvent(stage, attempt, rewrites, "", nil))
	})
	if err != nil {
		logf("regenerate job %s failed: %v", jobID, err)
		emit(stageEvent(StageFailed, rewrites, rewrites, "", []string{ReasonGenerationError}))
		emit(Event{Type: EventError, Data: map[string]string{"code": "regenerate_failed", "message": ReasonGenerationError}})
		return
	}
	if err := s.Store.ResolveLinkJob(ctx, jobID, lec.ID); err != nil {
		logf("resolve links of job %s: %v", jobID, err)
	}
	if lec.Status == "published" {
		emit(stageEvent(StagePublished, rewrites, rewrites, lec.ID, nil))
	} else { // 검수를 통과하지 못한 판은 needs_revision 으로 남고 발행되지 않는다.
		emit(stageEvent(StageFailed, rewrites, rewrites, "", []string{reasonReview}))
	}
	emit(Event{Type: EventDone, Data: map[string]any{
		"lectureId": lec.ID, "status": lec.Status, "flagged": lec.Flagged, "issues": []any{},
	}})
}
