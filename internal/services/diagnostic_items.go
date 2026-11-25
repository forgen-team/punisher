package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/lecture"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
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
	lang := s.Store.LocaleFor(ctx, userID) // 강의에 묶이지 않은 문항은 만든 사용자의 언어(i18n §1.2).
	dc.Reader = s.readerFor(ctx, userID, lang)
	anchors := s.pretestLecture(ctx, userID, plan, &dc)
	items, u, _, err := s.writeItemSet(ctx, apiKey, itemSetCall{model: s.diagnosticModel(plan.kind), attempts: plan.attempts,
		priority: priority, system: set.Text(prompts.BlockCheckPretest, s.interactionVars(ctx, userID, conceptID, lang)), language: lang},
		func(revisions []string) string { dc.Revisions = revisions; return prompts.Diagnostic(dc) },
		func(set check.ItemSet) []string { return check.ValidateDiagnostic(set, vc) })
	s.recordUsage(done, userID, "check_items", conceptID, u, err)
	var saved []store.CheckItem
	if err == nil {
		normalized := check.NormalizeDiagnostic(items)
		if anchors != nil {
			linked := keepLectureAnchors(normalized, anchors)
			logf("pretest %s: %d/%d items point to the lecture body", sessionID, linked, len(normalized))
		}
		saved, err = s.Store.SaveDiagnosticItems(done, store.DiagnosticItems{Language: lang, PromptHash: set.Hash(prompts.BlockCheckPretest),
			CreatedFor: userID, Model: u.model, PromptVersion: diagnosticPromptVersion(set), Items: normalized})
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

// pretestLecture gives a pretest of a stored lecture its body and anchors
// (research #2: predictions the lecture answers). It returns the anchor set
// (nil = no lecture: a pretest while an instant lecture is written, or another purpose).
func (s *Service) pretestLecture(ctx context.Context, userID string, plan diagnosticPlan, dc *prompts.DiagnosticContext) map[string]bool {
	if plan.kind != check.PurposePretest || plan.lectureID == "" {
		return nil
	}
	lec, err := s.Store.GetLecture(ctx, userID, plan.lectureID)
	if err != nil {
		logf("pretest lecture %s: %v (objectives only)", plan.lectureID, err)
		return nil
	}
	var c lecture.Content
	if err := json.Unmarshal(lec.Content, &c); err != nil {
		logf("pretest lecture %s: %v (objectives only)", plan.lectureID, err)
		return nil
	}
	anchors := map[string]bool{}
	for _, sec := range c.Sections {
		var b strings.Builder
		fmt.Fprintf(&b, "### [%s] %s\n", sec.Anchor, sec.Title)
		anchors[sec.Anchor] = true
		dc.Anchors = append(dc.Anchors, sec.Anchor)
		for _, blk := range sec.Blocks {
			if blk.Type == lecture.BlockRecall || blk.Type == lecture.BlockAnswer || blk.Type == lecture.BlockSummary {
				continue // 답이 그대로 적힌 블록은 사전 질문의 재료가 아니다
			}
			if blk.Anchor != "" && !anchors[blk.Anchor] {
				anchors[blk.Anchor] = true
				dc.Anchors = append(dc.Anchors, blk.Anchor)
			}
			writeCheckBlock(&b, blk)
		}
		dc.Lecture = append(dc.Lecture, b.String())
	}
	return anchors
}

// keepLectureAnchors clears item anchors outside the lecture and returns how
// many items point to it (G19: pretest questions the lecture answers).
func keepLectureAnchors(items []check.Item, anchors map[string]bool) int {
	linked := 0
	for i := range items {
		if anchors[items[i].Rubric.Anchor] {
			linked++
		} else {
			items[i].Rubric.Anchor = ""
		}
	}
	return linked
}
