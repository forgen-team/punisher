package services

import (
	"context"
	"encoding/json"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// checkState is a loaded session with its items and turns.
type checkState struct {
	sess  models.CheckSession
	items []store.CheckItem
	turns []models.CheckTurn
}

func (s *Service) loadCheckState(ctx context.Context, userID, id string) (checkState, error) {
	sess, err := s.Store.GetCheckSession(ctx, userID, id)
	if err != nil {
		return checkState{}, err
	}
	items, err := s.Store.CheckItemsByID(ctx, userID, sess.ItemIDs)
	if err != nil {
		return checkState{}, err
	}
	turns, err := s.Store.ListCheckTurns(ctx, userID, id)
	return checkState{sess: sess, items: items, turns: turns}, err
}

// current is the item under the cursor (nil when the session is not active).
func (st checkState) current() *store.CheckItem {
	if st.sess.Status != "active" || st.sess.Cursor >= len(st.items) {
		return nil
	}
	return &st.items[st.sess.Cursor]
}

func (st checkState) last() bool { return st.sess.Cursor >= len(st.items)-1 }

func (st checkState) pending() bool {
	if st.sess.Status == "preparing" {
		return true
	}
	for _, t := range st.turns {
		if t.Status == "pending" {
			return true
		}
	}
	return false
}

// itemProgress is one item's dialogue so far (failed turns and disputes excluded).
type itemProgress struct {
	answers    []models.CheckTurn // learner answers
	judgements []models.CheckTurn // tutor turns with a verdict
	lastTutor  *models.CheckTurn
	challenged bool
	done       bool
	outcome    string
}

func progressOf(turns []models.CheckTurn, itemID string) itemProgress {
	var p itemProgress
	for i := range turns {
		t := turns[i]
		if t.ItemID != itemID || t.Status == "failed" || t.Move == check.MoveDispute {
			continue
		}
		if t.Outcome != "" {
			p.done, p.outcome = true, t.Outcome
		}
		switch {
		case t.Role == "learner" && t.Move == check.MoveAnswer:
			p.answers = append(p.answers, t)
		case t.Role == "tutor":
			p.lastTutor = &turns[i]
			if t.Verdict != "" && t.Status == "ready" {
				p.judgements = append(p.judgements, t)
			}
			p.challenged = p.challenged || (t.Move == check.MoveChallenge && t.Status == "ready")
		}
	}
	return p
}

// awaitingAnswer: the tutor asked (ask, narrow, explain, challenge) and nothing
// is running; kind bounds the answers (a tutored review stops after narrowing).
func (p itemProgress) awaitingAnswer(kind string) bool {
	return !p.done && p.lastTutor != nil && p.lastTutor.Status == "ready" && !check.EndsItem(p.lastTutor.Move) &&
		len(p.answers) < check.MaxAnswersFor(kind) && (len(p.answers) == 0 || p.lastTutor.Seq > p.answers[len(p.answers)-1].Seq)
}

// canReveal: '그냥 알려줘' before the first answer or on the narrowing question (design §4.2).
func (p itemProgress) canReveal(kind string) bool {
	if !p.awaitingAnswer(kind) || p.challenged {
		return false
	}
	return len(p.answers) == 0 || (len(p.answers) == 1 && p.lastTutor.Move == check.MoveNarrow)
}

// result is the ItemResult of the item ending with verdict (the current judgement).
func (p itemProgress) result(verdict string, revealed bool) learning.ItemResult {
	first := verdict
	if len(p.judgements) > 0 {
		first = p.judgements[0].Verdict
	}
	if p.challenged { // 통합 문항의 반론 답: 문항 결과는 첫 답으로 정한다.
		return learning.ItemResult{FirstVerdict: first, FinalVerdict: first}
	}
	return learning.ItemResult{FirstVerdict: first, FinalVerdict: verdict, Aided: len(p.answers) > 1, Revealed: revealed}
}

func (s *Service) checkDetail(ctx context.Context, userID string, st checkState) (models.CheckDetail, error) {
	d := models.CheckDetail{Session: st.sess, Pending: st.pending(), Items: []models.CheckItemView{}, Turns: []models.CheckTurn{}}
	if d.Session.Error != "" {
		d.Session.Error = inquiryFailedMessage
	}
	objectives := []string{}
	for _, it := range st.items {
		p := progressOf(st.turns, it.ID)
		v := models.CheckItemView{ID: it.ID, ObjectiveID: it.ObjectiveID, Format: it.Format, Prompt: it.Prompt,
			Material: it.Material, Done: p.done, Outcome: p.outcome}
		if p.done || (st.sess.Grader == learning.GraderSelf && len(p.answers) > 0) {
			v.Answer, v.Must, v.Anchor = it.Rubric.Answer, it.Rubric.Must, it.Rubric.Anchor
		}
		d.Items = append(d.Items, v)
		if it.ObjectiveID != "" {
			objectives = append(objectives, it.ObjectiveID)
		}
	}
	for _, t := range st.turns {
		d.Turns = append(d.Turns, publicTurn(t))
	}
	if cur := st.current(); cur != nil {
		d.Current = currentView(st, *cur, d.Pending)
	}
	var err error
	d.Mastery, err = s.Store.ObjectivesMastery(ctx, userID, objectives)
	return d, err
}

func currentView(st checkState, it store.CheckItem, pending bool) *models.CheckCurrent {
	p := progressOf(st.turns, it.ID)
	c := &models.CheckCurrent{ItemID: it.ID, Answers: len(p.answers)}
	if pending || p.done {
		return c
	}
	if st.sess.Grader == learning.GraderSelf {
		c.CanAnswer, c.CanSelfGrade = len(p.answers) == 0, len(p.answers) > 0
		return c
	}
	c.CanAnswer, c.CanReveal = p.awaitingAnswer(st.sess.Kind), p.canReveal(st.sess.Kind)
	return c
}

// publicTurn hides the restore snapshot and raw failure text.
func publicTurn(t models.CheckTurn) models.CheckTurn {
	if t.Error != "" {
		t.Error = inquiryFailedMessage
	}
	if len(t.GradeRaw) > 0 && t.Verdict != "" {
		var g models.CheckGrade
		if err := json.Unmarshal(t.GradeRaw, &g); err != nil {
			logf("check turn %s: decode grade: %v", t.ID, err)
		} else {
			t.Grade = &g
		}
	}
	return t
}
