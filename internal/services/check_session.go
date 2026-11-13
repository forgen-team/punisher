package services

import (
	"context"
	"encoding/json"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// RetryCheck is '다시 시도': a failed session regenerates its item set; a
// failed tutor/dispute turn (the session's latest turn) runs again as a new pending turn.
func (s *Service) RetryCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if st.sess.Status == "failed" {
		return s.retryPrepare(ctx, userID, st)
	}
	if len(st.turns) == 0 || st.pending() {
		return models.CheckDetail{}, checkStateErr(st)
	}
	last := st.turns[len(st.turns)-1]
	if last.Role != "tutor" || last.Status != "failed" {
		return models.CheckDetail{}, ErrCheckState
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	dispute := last.Move == check.MoveDispute
	grader := learning.GraderLLM
	if dispute {
		grader = learning.GraderRegrade
	}
	turn, err := s.Store.StartCheckTurns(ctx, userID, id, nil,
		&store.TurnWrite{ItemID: last.ItemID, Role: "tutor", Move: last.Move, Status: "pending", Grader: grader}, maxPendingChecks, !dispute)
	if err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	runDetached("check retry", turn.ID, checkTurnTimeout, func(bg context.Context) error {
		if dispute {
			return s.runDispute(bg, apiKey, userID, id, turn.ID)
		}
		return s.runCheckTurn(bg, apiKey, userID, id, turn.ID)
	})
	return s.GetCheck(ctx, userID, id)
}

func (s *Service) retryPrepare(ctx context.Context, userID string, st checkState) (models.CheckDetail, error) {
	cl, err := s.loadCheckLecture(ctx, userID, st.sess.LectureID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if err := s.Store.SetCheckSessionStatus(ctx, userID, st.sess.ID, []string{"failed"}, "preparing", ""); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	if err := s.prepareCheck(ctx, apiKey, userID, st.sess.ID, cl); err != nil {
		if ferr := s.Store.SetCheckSessionStatus(ctx, userID, st.sess.ID, []string{"preparing"}, "failed", checkErrorText(err)); ferr != nil {
			logf("check %s: fail session after retry: %v", st.sess.ID, ferr)
		}
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, st.sess.ID)
}

// AbandonCheck is '그만하기': evidence already applied stays. A current item
// with a graded answer and nothing running ends with what it has (e.g. a
// first partial answer left before the narrowing question → unaided_partial).
func (s *Service) AbandonCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if item := st.current(); item != nil && !st.pending() {
		p := progressOf(st.turns, item.ID)
		if n := len(p.judgements); n > 0 && !p.done {
			last := p.judgements[n-1]
			turn := rewriteTurn(last)
			if err := s.finishItem(ctx, userID, st, *item, p.result(last.Verdict, false), false, turn, true); err != nil {
				return models.CheckDetail{}, checkBusy(err)
			}
			return s.GetCheck(ctx, userID, id)
		}
	}
	if err := s.Store.SetCheckSessionStatus(ctx, userID, id, []string{"preparing", "active"}, "abandoned", ""); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, id)
}

// rewriteTurn re-writes a ready judgement turn as the item's last (adds outcome + snapshot).
func rewriteTurn(t models.CheckTurn) store.TurnWrite {
	w := store.TurnWrite{ID: t.ID, ItemID: t.ItemID, Role: t.Role, Move: t.Move, Text: t.Text, Verdict: t.Verdict,
		Aided: t.Aided, Grader: t.Grader, Model: t.Model}
	if len(t.GradeRaw) > 0 {
		if err := json.Unmarshal(t.GradeRaw, &w.Grade.CheckGrade); err != nil {
			logf("check turn %s: decode grade: %v", t.ID, err)
		}
	}
	return w
}

// Relearn is '다시 배우기' on an objective (store.ReopenObjective) and returns its concept's mastery.
func (s *Service) Relearn(ctx context.Context, userID, objectiveID string) (models.ConceptMastery, error) {
	conceptID, err := s.Store.ReopenObjective(ctx, userID, objectiveID)
	if err != nil {
		return models.ConceptMastery{}, err
	}
	return s.Mastery(ctx, userID, conceptID)
}

// Mastery is the concept's objectives with levels, next review, stale and grader (GET /v1/mastery).
func (s *Service) Mastery(ctx context.Context, userID, conceptID string) (models.ConceptMastery, error) {
	list, err := s.Store.ConceptMastery(ctx, userID, conceptID)
	if err != nil {
		return models.ConceptMastery{}, err
	}
	settings, err := s.Store.GetLearningSettings(ctx, userID)
	if err != nil {
		logf("mastery target for %s: %v (using default)", userID, err)
	}
	out := models.ConceptMastery{ConceptID: conceptID, MasteryTarget: settings.MasteryTarget, Objectives: list}
	levels := make([]int, 0, len(list))
	for _, o := range list {
		levels = append(levels, o.Level)
		if o.UpdatedAt == nil || o.Best == learning.BestNone {
			out.UncheckedCount++
		}
	}
	out.Level = learning.ConceptLevel(levels)
	out.Mastered = learning.ConceptMastered(levels, out.MasteryTarget)
	return out, nil
}
