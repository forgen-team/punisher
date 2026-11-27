package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// TestSkipCheckItem: '건너뛰기' on a 확인하기 item ends it as skipped (no model
// answer shown, no evidence), asks the next item, and puts the objective in the
// review schedule at the nearest date. Only the current item, only before its
// first answer.
func TestSkipCheckItem(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	id := d.Session.ID
	d = waitIdle(t, s, user, id)
	if d.Current == nil || !d.Current.CanSkip {
		t.Fatalf("an asked item can be skipped: %+v", d.Current)
	}
	if _, err := s.SkipCheck(ctx, user, id, d.Items[1].ID); !errors.Is(err, ErrCheckState) {
		t.Fatalf("skipping another item = %v, want ErrCheckState", err)
	}

	got, err := s.SkipCheck(ctx, user, id, d.Items[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Session.Status != "active" || got.Session.Cursor != 1 || got.Current == nil || got.Current.ItemID != got.Items[1].ID {
		t.Fatalf("the session moves to the next item: %+v current=%+v", got.Session, got.Current)
	}
	if it := got.Items[0]; !it.Done || it.Outcome != learning.OutcomeSkipped || it.Answer != "" {
		t.Fatalf("skipped item = %+v (done, skipped, model answer hidden)", it)
	}
	if last := lastTurn(got); last.Move != "ask" || last.ItemID != got.Items[1].ID {
		t.Fatalf("the next item is asked: %+v", last)
	}
	var skipTurn models.CheckTurn
	for _, turn := range got.Turns {
		if turn.Move == "skip" {
			skipTurn = turn
		}
	}
	if skipTurn.Role != "learner" || skipTurn.Outcome != learning.OutcomeSkipped || skipTurn.Verdict != "" {
		t.Fatalf("the skip is recorded as the item's last turn: %+v", skipTurn)
	}
	if r := resultOf(t, got); len(r) != 1 || r[0].Outcome != learning.OutcomeSkipped {
		t.Fatalf("session result records the skip: %s", got.Session.Result)
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	want := learning.Due(0, now, nil)
	if a.Box != 0 || a.Lapses != 0 || a.Best != learning.BestNone || a.Level != learning.LevelUnchecked ||
		a.NextReviewAt == nil || !a.NextReviewAt.Equal(want) {
		t.Fatalf("skipped objective = %+v, want no evidence and the review at %v", a, want)
	}

	// 답을 보낸 뒤에는 건너뛸 수 없다.
	fake.push("tutor", tutorJSON("partial", "narrow", "어디에 남나요?", 0.9))
	if _, err := s.AnswerCheck(ctx, user, id, "메모리에 남아요", ""); err != nil {
		t.Fatal(err)
	}
	d = waitIdle(t, s, user, id)
	if d.Current == nil || d.Current.CanSkip {
		t.Fatalf("no skip after the first answer: %+v", d.Current)
	}
	if _, err := s.SkipCheck(ctx, user, id, d.Current.ItemID); !errors.Is(err, ErrCheckState) {
		t.Fatalf("skip after an answer = %v, want ErrCheckState", err)
	}
}

// TestSkipReviewKeepsLadder: skipping a due review keeps box, best and lapses
// (a failure would reset them) and moves the review to the nearest date.
func TestSkipReviewKeepsLadder(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	due := now.Add(-2 * time.Hour).Format(time.RFC3339Nano)
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, lapses,
		next_review_at, objective_text, objective_depth) SELECT ?, o.id, o.concept_id, 2, 'unaided', 1, ?, o.text, o.depth FROM objectives o WHERE o.id = ?`,
		user, due, f.objA); err != nil {
		t.Fatal(err)
	}
	ids := checkItemIDs(t, s, fake, user, f)
	id := activeSession(t, s, user, "review", learning.GraderSelf, ids[:2])

	got, err := s.SkipCheck(ctx, user, id, ids[0])
	if err != nil {
		t.Fatal(err)
	}
	if got.Session.Cursor != 1 || got.Current == nil || !got.Current.CanSkip {
		t.Fatalf("review moves to the next item: %+v %+v", got.Session, got.Current)
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	want := learning.Due(0, now, nil)
	if a.Box != 2 || a.Best != learning.BestUnaided || a.Lapses != 1 || a.NextReviewAt == nil || !a.NextReviewAt.Equal(want) {
		t.Fatalf("skipped review = %+v, want box 2 · unaided · lapses 1 · next %v", a, want)
	}
	got, err = s.SkipCheck(ctx, user, id, ids[1])
	if err != nil || got.Session.Status != "finished" {
		t.Fatalf("skipping the last item finishes the session: %+v (%v)", got.Session, err)
	}
	if n := len(resultOf(t, got)); n != 2 {
		t.Fatalf("result items = %d, want 2", n)
	}
}

// TestSkipEveryKind: the skip works in every answered session kind and is
// refused on the record-only diagnostics (they have '모르겠어요').
func TestSkipEveryKind(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	ids := checkItemIDs(t, s, fake, user, f)
	for _, tc := range []struct{ kind, grader string }{
		{"review", learning.GraderLLM}, {"synthesis", learning.GraderLLM}, {"known_verify", learning.GraderLLM},
	} {
		id := activeSession(t, s, user, tc.kind, tc.grader, []string{ids[2], ids[0]})
		got, err := s.SkipCheck(ctx, user, id, ids[2])
		if err != nil || got.Session.Cursor != 1 || got.Items[0].Outcome != learning.OutcomeSkipped {
			t.Fatalf("%s: skip = %+v (%v)", tc.kind, got.Session, err)
		}
	}
	id := activeSession(t, s, user, "pretest", "none", []string{ids[0]})
	d, err := s.GetCheck(ctx, user, id)
	if err != nil || d.Current == nil || d.Current.CanSkip {
		t.Fatalf("pretest offers no skip: %+v (%v)", d.Current, err)
	}
	if _, err := s.SkipCheck(ctx, user, id, ids[0]); !errors.Is(err, ErrCheckState) {
		t.Fatalf("pretest skip = %v, want ErrCheckState", err)
	}
}

// checkItemIDs generates the lecture's check items (objA, objB, integrative) through a 확인하기 session.
func checkItemIDs(t *testing.T, s *Service, fake *checkLLM, user string, f checkFixture) []string {
	t.Helper()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(context.Background(), user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	d = waitIdle(t, s, user, d.Session.ID)
	if _, err := s.AbandonCheck(context.Background(), user, d.Session.ID); err != nil {
		t.Fatal(err)
	}
	ids := make([]string, 0, len(d.Items))
	for _, it := range d.Items {
		ids = append(ids, it.ID)
	}
	return ids
}

// activeSession opens a session of the kind over the items, asking the first.
func activeSession(t *testing.T, s *Service, user, kind, grader string, ids []string) string {
	t.Helper()
	ctx := context.Background()
	sess, _, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: user, Kind: kind, Grader: grader,
		SessionKey: kind + ":skip:" + store.NewID()})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.ActivateCheckSession(ctx, user, sess.ID, ids, store.TurnWrite{ItemID: ids[0], Role: "tutor", Move: "ask", Text: "질문"}); err != nil {
		t.Fatal(err)
	}
	return sess.ID
}

func resultOf(t *testing.T, d models.CheckDetail) []store.ResultEntry {
	t.Helper()
	var r struct {
		Items []store.ResultEntry `json:"items"`
	}
	if err := json.Unmarshal(d.Session.Result, &r); err != nil {
		t.Fatal(err)
	}
	return r.Items
}
