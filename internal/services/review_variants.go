package services

import (
	"context"
	"errors"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// maxVariantsPerTick bounds the review variants written in one scheduler tick.
const maxVariantsPerTick = 10

// VariantReport is what one PregenerateReviewVariants run did.
type VariantReport struct {
	Generated int
	Failed    int
	// Skipped: already attempted today (idempotency key taken).
	Skipped int
}

// PregenerateReviewVariants writes, for the owner's objectives due by the end
// of tomorrow that have no unseen item, one review variant each (design §5 P2)
// — within the daily check budget (checkKey: UsageSinceChecks /
// PUNISHER_DAILY_CHECK_TOKEN_LIMIT), once per objective per day (generation_jobs key).
func (s *Service) PregenerateReviewVariants(ctx context.Context) (VariantReport, error) {
	var rep VariantReport
	owner := s.Cfg.OwnerUserID
	if owner == "" {
		return rep, nil
	}
	now := s.now()
	targets, err := s.Store.ReviewVariantTargets(ctx, owner, learning.KSTDayStart(now).AddDate(0, 0, 2), maxVariantsPerTick)
	if err != nil {
		return rep, err
	}
	for _, t := range targets {
		apiKey, err := s.checkKey(ctx, owner)
		if err != nil {
			return rep, err
		}
		key := "check_variant:" + owner + ":" + t.ObjectiveID + ":" + learning.KSTDay(now)
		job, created, err := s.Store.CreateJob(ctx, store.NewJob{IdempotencyKey: key, Kind: "check_items", UserID: owner,
			ConceptID: t.ConceptID, ObjectiveKey: key, Reason: "review variant"})
		if err != nil {
			return rep, err
		}
		if !created {
			rep.Skipped++
			continue
		}
		if err := s.Store.StartJob(ctx, job.ID); err != nil {
			return rep, err
		}
		if err := s.runReviewVariant(ctx, apiKey, owner, t, job.ID); err != nil {
			rep.Failed++
			logf("review variant %s: %v", t.ObjectiveID, err)
			if errors.Is(err, llm.ErrGenerationUnavailable) || errors.Is(err, llm.ErrRateLimited) {
				return rep, err
			}
			continue
		}
		rep.Generated++
	}
	return rep, nil
}

// runReviewVariant writes one review item for the target's objective from its
// lecture with the check.items prompt and validation, records the tokens on
// the job (kind check_items) and stores the item. Failure stores nothing.
func (s *Service) runReviewVariant(ctx context.Context, apiKey, userID string, t store.VariantTarget, jobID string) error {
	cl, err := s.loadCheckLecture(ctx, userID, t.LectureID)
	if err == nil {
		err = narrowToObjective(&cl, t.ObjectiveID)
	}
	if err != nil {
		if ferr := s.Store.FinishJob(ctx, jobID, usage{}.outcome("failed", "", err, "")); ferr != nil {
			logf("review variant %s: finish job: %v", t.ObjectiveID, ferr)
		}
		return err
	}
	set := s.UserPrompts(ctx, userID)
	ic, sc := checkItemsContext(cl)
	system := set.Text(prompts.BlockCheckItems, s.interactionVars(ctx, userID, cl.info.ID))
	seen := map[string][]string{t.ObjectiveID: t.Seen}
	items, u, tries, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.Cfg.LectureModel, attempts: checkAttempts,
		priority: llm.PriorityBackground, system: system},
		func(revisions []string) string { ic.Revisions = revisions; return prompts.CheckVariants(ic, seen) },
		func(set check.ItemSet) []string { return check.ValidateVariants(set, sc, []string{t.ObjectiveID}) })
	if err == nil {
		key := s.checkItemKey(ctx, userID, cl.lec.ID)
		key.Purpose = check.PurposeReview
		_, err = s.Store.SaveVariantItems(ctx, key, cl.info.ID, jobID, u.model, checkPromptVersion(set),
			check.NormalizeItems(items, cl.objectiveIDs()))
	}
	s.finishItemsJob(ctx, "review variant "+t.ObjectiveID, jobID, u, tries, err)
	return err
}

func narrowToObjective(cl *checkLecture, objectiveID string) error {
	for _, o := range cl.objectives {
		if o.ID == objectiveID {
			cl.objectives = []models.ObjectiveInfo{o}
			return nil
		}
	}
	return invalidCheck(CodeNoObjectives, "objective %s is not active in lecture %s", objectiveID, cl.lec.ID)
}
