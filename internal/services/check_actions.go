package services

import (
	"context"
	"errors"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// AnswerCheck stores the learner's answer to the current item. With the LLM
// grader the tutor's reply is written in the background (202, poll while
// pending); a self-graded session only records the answer (then self-grade).
func (s *Service) AnswerCheck(ctx context.Context, userID, id, text string) (models.CheckDetail, error) {
	text, ok := validText(text, MaxCheckAnswerRunes)
	if !ok {
		if text == "" {
			return models.CheckDetail{}, invalidCheck(CodeAnswerEmpty, "answer is required")
		}
		return models.CheckDetail{}, invalidCheck(CodeAnswerTooLong, "answer is too long (max %d characters)", MaxCheckAnswerRunes)
	}
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	item := st.current()
	if item == nil || st.pending() {
		return models.CheckDetail{}, checkStateErr(st)
	}
	p := progressOf(st.turns, item.ID)
	learner := &store.TurnWrite{ItemID: item.ID, Role: "learner", Move: check.MoveAnswer, Text: text, Aided: len(p.answers) > 0}
	if st.sess.Grader == learning.GraderSelf {
		if len(p.answers) > 0 || p.done {
			return models.CheckDetail{}, ErrCheckState
		}
		if _, err := s.Store.StartCheckTurns(ctx, userID, id, learner, nil, maxPendingChecks, true); err != nil {
			return models.CheckDetail{}, checkBusy(err)
		}
		return s.GetCheck(ctx, userID, id)
	}
	if !p.awaitingAnswer(st.sess.Kind) {
		return models.CheckDetail{}, ErrCheckState
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	tutor := &store.TurnWrite{ItemID: item.ID, Role: "tutor", Status: "pending", Grader: learning.GraderLLM}
	turn, err := s.Store.StartCheckTurns(ctx, userID, id, learner, tutor, maxPendingChecks, true)
	if err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	runDetached("check turn", turn.ID, checkTurnTimeout, func(bg context.Context) error {
		return s.runCheckTurn(bg, apiKey, userID, id, turn.ID)
	})
	return s.GetCheck(ctx, userID, id)
}

func checkStateErr(st checkState) error {
	if st.pending() && st.sess.Status == "active" {
		return ErrCheckBusy
	}
	return ErrCheckState
}

// RevealCheck is '그냥 알려줘': the model answer is shown and the item ends as failed (revealed).
func (s *Service) RevealCheck(ctx context.Context, userID, id string) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	item := st.current()
	if item == nil || st.pending() || st.sess.Grader != learning.GraderLLM {
		return models.CheckDetail{}, checkStateErr(st)
	}
	p := progressOf(st.turns, item.ID)
	if !p.canReveal(st.sess.Kind) {
		return models.CheckDetail{}, ErrCheckState
	}
	turn := store.TurnWrite{ItemID: item.ID, Role: "tutor", Move: check.MoveReveal, Text: item.Rubric.Answer,
		Verdict: learning.VerdictRevealed, Aided: len(p.answers) > 0, Grader: st.sess.Grader}
	if err := s.finishItem(ctx, userID, st, *item, p.result(learning.VerdictRevealed, true), false, turn, false); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, id)
}

var selfVerdicts = map[string]bool{learning.VerdictCorrect: true, learning.VerdictPartial: true, learning.VerdictIncorrect: true}

// SelfGradeCheck records the learner's own grade (P2 self-graded reviews):
// only after an answer was written, and it ends the item.
func (s *Service) SelfGradeCheck(ctx context.Context, userID, id, verdict string) (models.CheckDetail, error) {
	if !selfVerdicts[verdict] {
		return models.CheckDetail{}, invalidCheck(CodeInvalidVerdict, "verdict must be correct|partial|incorrect")
	}
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	item := st.current()
	if item == nil || st.sess.Grader != learning.GraderSelf {
		return models.CheckDetail{}, ErrCheckState
	}
	p := progressOf(st.turns, item.ID)
	if len(p.answers) == 0 || p.done {
		return models.CheckDetail{}, ErrCheckState
	}
	turn := store.TurnWrite{ItemID: item.ID, Role: "learner", Move: check.MoveSelfGrade, Verdict: verdict, Grader: learning.GraderSelf}
	r := learning.ItemResult{FirstVerdict: verdict, FinalVerdict: verdict}
	if err := s.finishItem(ctx, userID, st, *item, r, false, turn, false); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, id)
}

var disputable = map[string]bool{learning.OutcomeAided: true, learning.OutcomeFailed: true,
	learning.OutcomeHeld: true, learning.OutcomeUnaidedPartial: true}

// DisputeCheck is '내 답이 맞아요' on an item's last judgement: another model
// regrades the first answer from the rubric only (the reason is kept as data).
// Once per item, while the session is open or within 24h of its end.
func (s *Service) DisputeCheck(ctx context.Context, userID, id, turnID, reason string) (models.CheckDetail, error) {
	if len([]rune(reason)) > MaxDisputeRunes {
		return models.CheckDetail{}, invalidCheck(CodeReasonTooLong, "reason too long (max %d characters)", MaxDisputeRunes)
	}
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	if st.pending() || !s.disputeOpen(st.sess) {
		return models.CheckDetail{}, checkStateErr(st)
	}
	judged, item, err := judgedTurn(st, turnID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	apiKey, err := s.checkKey(ctx, userID)
	if err != nil {
		return models.CheckDetail{}, err
	}
	learner := &store.TurnWrite{ItemID: item.ID, Role: "learner", Move: check.MoveDispute, Text: reason}
	tutor := &store.TurnWrite{ItemID: judged.ItemID, Role: "tutor", Move: check.MoveDispute, Status: "pending", Grader: learning.GraderRegrade}
	turn, err := s.Store.StartCheckTurns(ctx, userID, id, learner, tutor, maxPendingChecks, false)
	if err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	runDetached("check dispute", turn.ID, checkTurnTimeout, func(bg context.Context) error {
		return s.runDispute(bg, apiKey, userID, id, turn.ID)
	})
	return s.GetCheck(ctx, userID, id)
}

func (s *Service) disputeOpen(sess models.CheckSession) bool {
	switch sess.Status {
	case "active":
		return true
	case "finished", "abandoned":
		return sess.FinishedAt != nil && s.now().Sub(*sess.FinishedAt) < disputeWindow
	}
	return false
}

// judgedTurn validates the dispute target: an objective item's last
// judgement with a disputable outcome, not disputed before.
func judgedTurn(st checkState, turnID string) (models.CheckTurn, store.CheckItem, error) {
	var judged *models.CheckTurn
	for i, t := range st.turns {
		if t.ID == turnID {
			judged = &st.turns[i]
		}
	}
	if judged == nil {
		return models.CheckTurn{}, store.CheckItem{}, store.ErrNotFound
	}
	item, ok := itemByID(st.items, judged.ItemID)
	if !ok || item.ObjectiveID == "" || !disputable[judged.Outcome] || judged.Verdict == learning.VerdictRevealed {
		return *judged, item, ErrCheckState
	}
	for _, t := range st.turns {
		if t.ItemID == item.ID && t.Role == "tutor" && t.Move == check.MoveDispute && t.Status != "failed" {
			return *judged, item, ErrCheckState // 이의는 문항당 1회
		}
	}
	return *judged, item, nil
}

// disputeTarget finds, for a pending dispute turn, the item, its judged turn and the first answer.
func disputeTarget(st checkState, turnID string) (store.CheckItem, models.CheckTurn, string, error) {
	var itemID string
	for _, t := range st.turns {
		if t.ID == turnID {
			itemID = t.ItemID
		}
	}
	item, ok := itemByID(st.items, itemID)
	if !ok {
		return item, models.CheckTurn{}, "", store.ErrNotFound
	}
	p := progressOf(st.turns, item.ID)
	var judged models.CheckTurn
	for _, t := range st.turns {
		if t.ItemID == item.ID && t.Outcome != "" {
			judged = t
		}
	}
	if judged.ID == "" || len(p.answers) == 0 {
		return item, judged, "", errors.New("nothing to regrade")
	}
	return item, judged, p.answers[0].Text, nil
}

func itemByID(items []store.CheckItem, id string) (store.CheckItem, bool) {
	for _, it := range items {
		if it.ID == id {
			return it, true
		}
	}
	return store.CheckItem{}, false
}
