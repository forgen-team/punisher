package services

import (
	"context"
	"errors"
	"strings"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/prompts"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// runDiagnosticDetached writes the plan's items in the background.
func (s *Service) runDiagnosticDetached(apiKey, userID, sessionID string, plan diagnosticPlan) {
	runDetached("diagnostic items", sessionID, diagnosticTimeout, func(bg context.Context) error {
		return s.runDiagnosticItems(bg, apiKey, userID, sessionID, plan)
	}, s.failPreparing(userID, sessionID, ""))
}

// runDiagnosticItems writes the plan's items with the check.pretest block
// (diagnosticModel), records the tokens (kind check_items), stores the items and
// asks the first one. Failure fails the session, never stores items.
func (s *Service) runDiagnosticItems(ctx context.Context, apiKey, userID, sessionID string, plan diagnosticPlan) error {
	done := context.WithoutCancel(ctx)
	set := s.UserPrompts(ctx, userID)
	conceptID := plan.conceptID
	if conceptID == "" && len(plan.objectives) > 0 {
		conceptID = conceptOf(plan.objectives[0].ID)
	}
	dc := prompts.DiagnosticContext{Purpose: plan.kind, Objectives: plan.objectives, Count: check.PretestCount(len(plan.objectives))}
	vc := check.DiagnosticContext{Purpose: plan.kind, ObjectiveIDs: plan.objectiveIDs()}
	priority := llm.PriorityInteractive
	if plan.kind == check.PurposePretest { // 예측 질문은 강의를 기다리는 동안 나와야 하므로 짧은 호출 자리를 쓴다.
		priority = llm.ShortUnlessBackground(ctx)
	}
	items, u, _, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.diagnosticModel(plan.kind), attempts: plan.attempts,
		priority: priority, system: set.Text(prompts.BlockCheckPretest, s.interactionVars(ctx, userID, conceptID))},
		func(revisions []string) string { dc.Revisions = revisions; return prompts.Diagnostic(dc) },
		func(set check.ItemSet) []string { return check.ValidateDiagnostic(set, vc) })
	s.recordUsage(done, userID, "check_items", conceptID, u, err)
	var saved []store.CheckItem
	if err == nil {
		saved, err = s.Store.SaveDiagnosticItems(done, store.DiagnosticItems{PromptHash: set.Hash(prompts.BlockCheckPretest),
			CreatedFor: userID, Model: u.model, PromptVersion: diagnosticPromptVersion(set), Items: check.NormalizeDiagnostic(items)})
	}
	if err == nil {
		saved = orderByObjectives(saved, plan.objectiveIDs())
		ids := make([]string, 0, len(saved))
		for _, it := range saved {
			ids = append(ids, it.ID)
		}
		ask := store.TurnWrite{ItemID: saved[0].ID, Role: "tutor", Move: check.MoveAsk, Text: saved[0].Prompt}
		_, err = s.Store.ActivateCheckSession(done, userID, sessionID, ids, ask)
	}
	if err != nil {
		if serr := s.Store.SetCheckSessionStatus(done, userID, sessionID, []string{"preparing"}, "failed", checkErrorText(err)); serr != nil && !errors.Is(serr, store.ErrCheckClosed) {
			logf("diagnostic %s: fail session: %v", sessionID, serr)
		}
	}
	return err
}

// conceptOf is the concept of an objective id (objective ids are "<concept>#<n>").
func conceptOf(objectiveID string) string {
	concept, _, _ := strings.Cut(objectiveID, "#")
	return concept
}

// orderByObjectives keeps the plan's asking order (placement easy → hard);
// items of the same objective keep their generated order.
func orderByObjectives(items []store.CheckItem, objectiveIDs []string) []store.CheckItem {
	out := make([]store.CheckItem, 0, len(items))
	for _, id := range objectiveIDs {
		for _, it := range items {
			if it.ObjectiveID == id {
				out = append(out, it)
			}
		}
	}
	return out
}

func diagnosticPromptVersion(set prompts.Set) string {
	if h := set.Hash(prompts.BlockCheckPretest); h != "" {
		return prompts.CheckVersion + "+u:" + h
	}
	return prompts.CheckVersion
}
