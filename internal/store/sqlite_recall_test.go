package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/forgen-team/punisher/migrations"
)

// TestAddRecallReviewHold: with the review backlog over its limit a recall's
// objective waits in the queue (no date) like a new check's, and the item is
// still added; the lecture_reviews.quality default reads as no record.
func TestAddRecallReviewHold(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	lectureID := insertLecture(t, s, conceptID, objectiveID)
	_, err := s.c().exec(ctx, `UPDATE lectures SET requested_by = $2 WHERE id = $1`, lectureID, testUser)
	if err != nil {
		t.Fatal(err)
	}
	itemID, scheduled, err := s.AddRecallReview(ctx, RecallReview{UserID: testUser, LectureID: lectureID, ConceptID: conceptID,
		ObjectiveID: objectiveID, Anchor: "principles-9", Question: "q?", Key: "k", Due: time.Now().Add(24 * time.Hour), Hold: true})
	if err != nil || itemID == "" || !scheduled {
		t.Fatalf("add = %q %v %v", itemID, scheduled, err)
	}
	var queued bool
	var next *string
	if err := s.c().row(ctx, `SELECT queued, next_review_at FROM user_objective_mastery WHERE user_id = $1 AND objective_id = $2`,
		testUser, objectiveID).Scan(&queued, &next); err != nil {
		t.Fatal(err)
	}
	if !queued || next != nil {
		t.Fatalf("held recall waits: queued=%v next=%v", queued, next)
	}
	var promptHash, format, purpose string
	if err := s.c().row(ctx, `SELECT prompt_hash, format, purpose FROM check_items WHERE id = $1`, itemID).Scan(&promptHash, &format, &purpose); err != nil ||
		promptHash != RecallPromptHash || format != "recall" || purpose != "review" {
		t.Fatalf("item = %s %s %s %v", promptHash, format, purpose, err)
	}
	if recs, err := s.QualityRecords(ctx); err != nil || len(recs) != 0 {
		t.Fatalf("no quality record yet: %+v %v", recs, err)
	}
}

// TestUpgradeToV030: reviews and exercise attempts written before 030 keep
// working; the new columns read as "no quality record" and "no confidence".
func TestUpgradeToV030(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "030")); err != nil {
		t.Fatalf("pre-030 migrate: %v", err)
	}
	conceptID, objectiveID := masteryFixture(t, old, testUser)
	lectureID := insertLecture(t, old, conceptID, objectiveID)
	if _, err := old.c().exec(ctx, `INSERT INTO lecture_reviews (id, lecture_id, verdict) VALUES ('r1', $1, 'pass')`, lectureID); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO lecture_exercise_attempts (id, user_id, lecture_id, anchor, content_hash, kind, status)
		VALUES ('a1', $1, $2, 'principles-1', 'h', 'exercise', 'done')`, testUser, lectureID); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("upgrade migrate: %v", err)
	}
	a, err := s.GetExerciseAttempt(ctx, testUser, "a1")
	if err != nil || a.Confidence != "" || a.ReviewItemID != "" {
		t.Fatalf("old attempt = %+v %v", a, err)
	}
	var q string
	if err := s.c().row(ctx, `SELECT quality FROM lecture_reviews WHERE id = 'r1'`).Scan(&q); err != nil || q != "{}" {
		t.Fatalf("old review quality = %q %v", q, err)
	}
}

func TestRepeatedMisconceptions(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	stmts := []string{
		`INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt) VALUES ('it', '` + conceptID + `', '` + objectiveID + `', 'check', 'recall', 'q')`,
		`INSERT INTO check_sessions (id, user_id, kind, session_key, status) VALUES ('ss', '` + testUser + `', 'check', 'k', 'finished')`,
	}
	for i, m := range []string{"인덱스가 유니크를 보장한다", "인덱스가 유니크를 보장한다", "한 번만 나온 오해"} {
		stmts = append(stmts, `INSERT INTO check_turns (id, session_id, user_id, item_id, seq, role, grade) VALUES ('t`+string(rune('0'+i))+`', 'ss', '`+testUser+`', 'it', `+string(rune('1'+i))+`, 'tutor', '{"misconception":"`+m+`"}')`)
	}
	for _, q := range stmts {
		if _, err := s.c().exec(ctx, q); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.RepeatedMisconceptions(ctx, testUser, conceptID, 2, 3)
	if err != nil || len(got) != 1 || got[0] != "인덱스가 유니크를 보장한다" {
		t.Fatalf("repeated = %q %v", got, err)
	}
}
