package services

import (
	"context"
	"encoding/json"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// checkState is a loaded session with its items and turns.
type checkState struct {
	sess  models.CheckSession
	items []store.CheckItem
	turns []models.CheckTurn
}

func (s *Service) loadCheckState(ctx context.Context, userID, id string) (checkState, error) {
	if !validID(id) {
		return checkState{}, store.ErrNotFound
	}
	snap, err := s.Store.LoadCheckSnapshot(ctx, userID, id) // 세션·문항·턴을 한 스냅숏으로(커서와 판정이 어긋나 보이지 않게)
	return checkState{sess: snap.Session, items: snap.Items, turns: snap.Turns}, err
}

// current is the item under the cursor (nil when the session is not active).
func (st checkState) current() *store.CheckItem {
	if st.sess.Status != "active" || st.sess.Cursor >= len(st.items) {
		return nil
	}
	return &st.items[st.sess.Cursor]
}

func (st checkState) last() bool { return st.sess.Cursor >= len(st.items)-1 }

// itemKind is the kind the dialogue rules of an item follow: the session kind,
// except a known_verify item mixed into a review (one answer, no narrowing).
func (st checkState) itemKind(it store.CheckItem) string {
	if it.Purpose == check.PurposeKnownVerify {
		return check.PurposeKnownVerify
	}
	return st.sess.Kind
}

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
		// 건너뛴 문항은 모범답을 보이지 않는다(공개하면 '그냥 알려줘'를 벌 없이 쓰는 길이 된다).
		revealed := p.done && p.outcome != learning.OutcomeSkipped
		if !check.Recorded(st.sess.Kind) && (revealed || (st.sess.Grader == learning.GraderSelf && len(p.answers) > 0)) {
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
	if d.Mastery, err = s.Store.ObjectivesMastery(ctx, userID, objectives); err != nil {
		return d, err
	}
	if d.MisconceptionTurnIDs, err = s.Store.MisconceptionTurnIDs(ctx, userID, st.sess.ID); err != nil {
		return d, err
	}
	if err = s.attachLectureSummaries(ctx, userID, st.sess.Kind, d.Items, st.items); err != nil {
		return d, err
	}
	switch st.sess.Kind {
	case check.PurposeCheck:
		d.Predictions, err = s.Store.PretestAnswers(ctx, userID, objectives)
	case check.PurposePlacement:
		d.Placement = s.placementResult(ctx, userID, st)
	}
	return d, err
}

// attachLectureSummaries sets each done item's lecture summary (§6 '이 강의
// 한 장 요약'): check results and review items only — never before the answer,
// and never on diagnostics (a pretest comes before the lecture). A review mixes
// lectures, so the summary is per item.
func (s *Service) attachLectureSummaries(ctx context.Context, userID, kind string, views []models.CheckItemView, items []store.CheckItem) error {
	if kind != check.PurposeCheck && kind != check.PurposeReview {
		return nil
	}
	ids := []string{}
	seen := map[string]bool{}
	for i, it := range items {
		if views[i].Done && it.LectureID != "" && !seen[it.LectureID] {
			seen[it.LectureID] = true
			ids = append(ids, it.LectureID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	summaries, err := s.Store.LectureSummaries(ctx, userID, ids)
	if err != nil {
		return err
	}
	for i, it := range items {
		if views[i].Done {
			views[i].LectureSummary = summaries[it.LectureID]
		}
	}
	return nil
}

func currentView(st checkState, it store.CheckItem, pending bool) *models.CheckCurrent {
	p := progressOf(st.turns, it.ID)
	c := &models.CheckCurrent{ItemID: it.ID, Answers: len(p.answers)}
	if pending || p.done {
		return c
	}
	if st.sess.Grader == learning.GraderSelf || check.Recorded(st.sess.Kind) {
		c.CanAnswer, c.CanSelfGrade = len(p.answers) == 0, len(p.answers) > 0 && !check.Recorded(st.sess.Kind)
	} else {
		kind := st.itemKind(it)
		c.CanAnswer, c.CanReveal = p.awaitingAnswer(kind), p.canReveal(kind)
	}
	// 건너뛰기는 첫 답 전에만(답한 뒤 건너뛰면 판정을 피하는 길이 된다). 예측·배치는 '모르겠어요'가 있다.
	c.CanSkip = c.CanAnswer && len(p.answers) == 0 && !check.Recorded(st.sess.Kind)
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
