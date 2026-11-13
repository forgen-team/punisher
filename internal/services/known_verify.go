package services

import (
	"context"
	"errors"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// MarkKnown is '이미 앎' (POST /v1/concepts/{id}/known). queueVerify (bulk
// '이미 앎' from the onboarding) also puts the concept's objectives without
// evidence into the verification queue mixed into daily reviews (design §5 P3).
func (s *Service) MarkKnown(ctx context.Context, userID, conceptID string, queueVerify bool) (string, error) {
	state, err := s.Store.SetConceptState(ctx, userID, conceptID, learning.Known)
	if err != nil || !queueVerify || state != learning.Known {
		return state, err
	}
	_, err = s.Store.QueueKnownVerify(ctx, userID, conceptID)
	return state, err
}

// PregenerateKnownVerify writes, for the owner, the known_verify items of the
// next queued concept when fewer than a day's verifications are ready — once
// per concept per day (generation_jobs key), within the daily check budget.
func (s *Service) PregenerateKnownVerify(ctx context.Context) (VariantReport, error) {
	var rep VariantReport
	owner := s.Cfg.OwnerUserID
	if owner == "" {
		return rep, nil
	}
	ready, err := s.Store.KnownVerifyPicks(ctx, owner, learning.KnownVerifyPerDay)
	if err != nil || len(ready) >= learning.KnownVerifyPerDay {
		return rep, err
	}
	targets, err := s.Store.KnownVerifyTargets(ctx, owner, 1)
	if err != nil {
		return rep, err
	}
	for _, t := range targets {
		apiKey, err := s.checkKey(ctx, owner)
		if err != nil {
			return rep, err
		}
		key := "known_verify:" + owner + ":" + t.ConceptID + ":" + learning.KSTDay(s.now())
		job, created, err := s.Store.CreateJob(ctx, store.NewJob{IdempotencyKey: key, Kind: "check_items", UserID: owner,
			ConceptID: t.ConceptID, ObjectiveKey: key, Reason: "known verify"})
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
		if err := s.runKnownVerifyItems(ctx, apiKey, owner, t, job.ID); err != nil {
			rep.Failed++
			logf("known verify items %s: %v", t.ConceptID, err)
			if errors.Is(err, llm.ErrGenerationUnavailable) || errors.Is(err, llm.ErrRateLimited) {
				return rep, err
			}
			continue
		}
		rep.Generated++
	}
	return rep, nil
}

func (s *Service) runKnownVerifyItems(ctx context.Context, apiKey, userID string, t store.KnownVerifyTarget, jobID string) error {
	objectives, err := s.diagnosticObjectives(ctx, userID, t.ConceptID, t.ObjectiveIDs)
	if err != nil {
		if ferr := s.Store.FinishJob(ctx, jobID, usage{}.outcome("failed", "", err, "")); ferr != nil {
			logf("known verify %s: finish job: %v", t.ConceptID, ferr)
		}
		return err
	}
	set := s.UserPrompts(ctx, userID)
	dc := prompts.DiagnosticContext{Purpose: check.PurposeKnownVerify, Objectives: objectives}
	vc := check.DiagnosticContext{Purpose: check.PurposeKnownVerify, ObjectiveIDs: diagnosticPlan{objectives: objectives}.objectiveIDs()}
	items, u, tries, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.fastModel(), attempts: checkAttempts,
		priority: llm.PriorityBackground, system: set.Text(prompts.BlockCheckPretest, s.interactionVars(ctx, userID, t.ConceptID))},
		func(revisions []string) string { dc.Revisions = revisions; return prompts.Diagnostic(dc) },
		func(set check.ItemSet) []string { return check.ValidateDiagnostic(set, vc) })
	status := "succeeded"
	if err != nil {
		status = "failed"
	}
	o := u.outcome(status, "", err, "")
	o.Attempts = tries
	if ferr := s.Store.FinishJob(context.WithoutCancel(ctx), jobID, o); ferr != nil {
		logf("known verify %s: finish job: %v", t.ConceptID, ferr)
	}
	if err != nil {
		return err
	}
	_, err = s.Store.SaveDiagnosticItems(ctx, store.DiagnosticItems{PromptHash: set.Hash(prompts.BlockCheckPretest), CreatedFor: userID,
		JobID: jobID, Model: u.model, PromptVersion: diagnosticPromptVersion(set), Items: check.NormalizeDiagnostic(items)})
	return err
}
