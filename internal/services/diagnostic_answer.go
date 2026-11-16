package services

import (
	"context"
	"strings"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// diagnosticAnswer is a learner answer on a pretest / placement item: a one-line
// prediction (text), an option (choice) or '모르겠어요' (dontKnow).
type diagnosticAnswer struct {
	text     string
	choice   *int
	dontKnow bool
}

// AnswerChoice records an option or '모르겠어요' on the current pretest / placement item.
func (s *Service) AnswerChoice(ctx context.Context, userID, id string, choice *int, dontKnow bool) (models.CheckDetail, error) {
	st, err := s.loadCheckState(ctx, userID, id)
	if err != nil {
		return models.CheckDetail{}, err
	}
	item := st.current()
	if item == nil || st.pending() || !check.Recorded(st.sess.Kind) {
		return models.CheckDetail{}, checkStateErr(st)
	}
	return s.recordAnswer(ctx, userID, st, *item, diagnosticAnswer{choice: choice, dontKnow: dontKnow})
}

// recordAnswer stores the answer as the item's last turn (outcome recorded, no
// feedback, no evidence — design §5 P3) and asks the next item. A placement
// choice is graded by code (correct / incorrect) for the "이미 앎 후보".
func (s *Service) recordAnswer(ctx context.Context, userID string, st checkState, item store.CheckItem, a diagnosticAnswer) (models.CheckDetail, error) {
	p := progressOf(st.turns, item.ID)
	if p.done || len(p.answers) > 0 {
		return models.CheckDetail{}, ErrCheckState
	}
	turn := store.TurnWrite{ItemID: item.ID, Role: "learner", Move: check.MoveAnswer, Text: a.text}
	choices := check.Choices(item.Material)
	switch {
	case a.dontKnow:
		turn.Move, turn.Text = check.MoveDontKnow, ""
	case item.Format == check.FormatChoice && a.choice == nil:
		return models.CheckDetail{}, invalidCheck(CodeChoiceRequired, "choose an option or dontKnow")
	case item.Format == check.FormatChoice && (*a.choice < 0 || *a.choice >= len(choices)):
		return models.CheckDetail{}, invalidCheck(CodeInvalidChoice, "choice must be 0~%d", len(choices)-1)
	case item.Format == check.FormatChoice:
		turn.Text, turn.Grade.Choice = choices[*a.choice], a.choice
	case a.choice != nil:
		return models.CheckDetail{}, invalidCheck(CodeInvalidChoice, "this item takes a text answer")
	}
	if st.sess.Kind == check.PurposePlacement {
		turn.Verdict = learning.VerdictIncorrect
		if c := item.Rubric.Correct; c != nil && a.choice != nil && !a.dontKnow && *c == *a.choice {
			turn.Verdict = learning.VerdictCorrect
		}
	}
	if err := s.finishItem(ctx, userID, st, item, learning.ItemResult{}, false, turn, false); err != nil {
		return models.CheckDetail{}, checkBusy(err)
	}
	return s.GetCheck(ctx, userID, st.sess.ID)
}

// placementResult: the concepts whose placement item was answered right ("이미 앎 후보").
func (s *Service) placementResult(ctx context.Context, userID string, st checkState) *models.PlacementResult {
	out := &models.PlacementResult{ChapterID: chapterOfKey(st.sess.SessionKey), Candidates: []models.PlacementCandidate{}}
	seen := map[string]bool{}
	for _, t := range st.turns {
		if t.Role != "learner" || t.Verdict != learning.VerdictCorrect {
			continue
		}
		item, ok := itemByID(st.items, t.ItemID)
		if !ok || seen[item.ConceptID] {
			continue
		}
		seen[item.ConceptID] = true
		name := item.ConceptID
		if info, err := s.Store.ConceptInfo(ctx, userID, item.ConceptID); err == nil {
			name = info.Name
		} else {
			logf("placement candidate %s: %v", item.ConceptID, err)
		}
		out.Candidates = append(out.Candidates, models.PlacementCandidate{ConceptID: item.ConceptID, ConceptName: name})
	}
	return out
}

func chapterOfKey(key string) string {
	chapterID, _ := strings.CutPrefix(key, "placement:")
	return chapterID
}
