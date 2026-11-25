package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/store"
)

// TestSelfGradedBlocks: a pause prediction, a recall and a summary recall are
// graded by the learner without an LLM call; a recall puts its question into
// the review bank and its objective's first review a day later — no mastery
// evidence — and leaves an objective that already has a review date alone.
func TestSelfGradedBlocks(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 9, 0, 0, 0, time.UTC)
	s.Now = func() time.Time { return now }
	content := `{"schemaVersion":1,"title":"t","sections":[
		{"anchor":"walkthrough","kind":"walkthrough","title":"따라가기","blocks":[
			{"anchor":"walkthrough-1","type":"code","lang":"sql","text":"SELECT 1;"},
			{"anchor":"walkthrough-2","type":"pause","text":"무엇이 나올까요?"},
			{"anchor":"walkthrough-3","type":"log","lang":"text","text":"1"},
			{"anchor":"walkthrough-4","type":"recall","objective":"` + f.objB + `","text":"방금 본 쿼리의 결과는?\n떠올릴 것: 1 한 행"},
			{"anchor":"walkthrough-5","type":"recall","text":"두 번째 질문?\n떠올릴 것: 답"}]},
		{"anchor":"judgment_next","kind":"judgment_next","title":"판단","blocks":[
			{"anchor":"judgment_next-1","type":"summary","text":"- a\n- b\n- c"}]}]}`
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lectures SET content = ? WHERE id = ?`, content, f.lectureID); err != nil {
		t.Fatal(err)
	}
	lec, err := s.Store.GetLecture(ctx, user, f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	pauseHash := BlockContentHash("무엇이 나올까요?", "1")
	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-2", ContentHash: pauseHash, Verdict: "remembered"}); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("a pause takes correct|incorrect, got %v", err)
	}
	p, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-2", ContentHash: pauseHash, Answer: "1",
		Verdict: learning.ExerciseIncorrect, Confidence: learning.ConfidenceSure})
	if err != nil || p.Kind != learning.AttemptPause || p.Status != store.AttemptDone || p.Confidence != learning.ConfidenceSure {
		t.Fatalf("pause = %+v %v", p, err)
	}

	recallHash := BlockContentHash("방금 본 쿼리의 결과는?\n떠올릴 것: 1 한 행", "")
	r, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-4", ContentHash: recallHash, Verdict: learning.RecallForgot})
	if err != nil || r.Kind != learning.AttemptRecall || r.ReviewItemID == "" {
		t.Fatalf("recall = %+v %v", r, err)
	}
	var box int
	var best string
	var next *string
	if err := testDB(s).QueryRowContext(ctx, `SELECT box, best, next_review_at FROM user_objective_mastery WHERE user_id = ? AND objective_id = ?`,
		user, f.objB).Scan(&box, &best, &next); err != nil {
		t.Fatal(err)
	}
	if box != 0 || best != "none" || next == nil {
		t.Fatalf("recall is not evidence: box=%d best=%s next=%v", box, best, next)
	}
	if due, _ := time.Parse(time.RFC3339Nano, *next); !due.Equal(now.Add(24 * time.Hour)) {
		t.Fatalf("first review a day later, got %s", *next)
	}
	items, err := s.Store.ReviewItems(ctx, user, []string{f.objB})
	if err != nil || items[f.objB].Item.ID != r.ReviewItemID || items[f.objB].Item.Prompt != "방금 본 쿼리의 결과는?" {
		t.Fatalf("the recall is the objective's review item: %+v %v", items, err)
	}
	again, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-4", ContentHash: recallHash, Verdict: learning.RecallRemembered})
	if err != nil || again.ReviewItemID != r.ReviewItemID {
		t.Fatalf("the same recall reuses its item: %+v %v", again, err)
	}

	// An objective already in the review schedule keeps its date.
	later := now.Add(10 * 24 * time.Hour).Format(time.RFC3339Nano)
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at)
		VALUES (?, ?, ?, 2, 'unaided', ?)`, user, f.objA, f.conceptID, later); err != nil {
		t.Fatal(err)
	}
	other := BlockContentHash("두 번째 질문?\n떠올릴 것: 답", "")
	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-5", ContentHash: other, Verdict: learning.ExercisePartial}); err != nil {
		t.Fatal(err)
	}
	var kept string
	if err := testDB(s).QueryRowContext(ctx, `SELECT next_review_at FROM user_objective_mastery WHERE user_id = ? AND objective_id = ?`, user, f.objA).Scan(&kept); err != nil || kept != later {
		t.Fatalf("scheduled objective must keep its date: %s %v", kept, err)
	}

	sumHash := BlockContentHash("- a\n- b\n- c", "")
	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "judgment_next-1", ContentHash: sumHash, Answer: " "}); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("a summary recall needs text, got %v", err)
	}
	sum, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "judgment_next-1", ContentHash: sumHash, Answer: "내가 떠올린 세 가지"})
	if err != nil || sum.Verdict != learning.SummaryWritten || sum.Answer != "" {
		t.Fatalf("summary keeps only that it was written: %+v %v", sum, err)
	}
	if fake.count("tutor") != 0 {
		t.Fatal("self-graded blocks never call the LLM")
	}

	// A confident wrong prediction marks its block for the regeneration (the
	// section stays unflagged: 1 miss of 3 attempts is under half).
	lines := s.signalLines(ctx, user, lec)
	if len(lines) != 1 || lines[0] != `따라가기 · "무엇이 나올까요?" — 확신하고 예측했는데 틀림` {
		t.Fatalf("block-level signals = %q", lines)
	}
}
