package services

import (
	"context"
	"errors"
	"testing"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// TestReviewConfidenceAndEase: a review answered 'sure' and self-graded wrong is
// a confident error (the outcome is still failed), the objective's ease drops by
// 0.3; a 'sure' correct answer raises it by 0.1. Turning adaptive intervals off
// resets ease to 2.0 and keeps it there.
func TestReviewConfidenceAndEase(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f) // box 1 with no confidence chosen → ease 2.05
	if got := ease(t, s, user, f.objA); got != 2.05 {
		t.Fatalf("ease after the check = %v, want 2.05", got)
	}
	dueNow(t, s, user, f.objA)
	dueNow(t, s, user, f.objB)
	d, _, err := s.StartReview(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	id := d.Session.ID
	if _, err := s.AnswerCheck(ctx, user, id, "x", "maybe"); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("unknown confidence must be rejected, got %v", err)
	}
	first, second := d.Items[0].ObjectiveID, d.Items[1].ObjectiveID
	if _, err := s.AnswerCheck(ctx, user, id, "모르겠어요", "sure"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SelfGradeCheck(ctx, user, id, "incorrect"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AnswerCheck(ctx, user, id, "메모리에 남아요", "sure"); err != nil {
		t.Fatal(err)
	}
	done, err := s.SelfGradeCheck(ctx, user, id, "correct")
	if err != nil {
		t.Fatal(err)
	}
	if got := ease(t, s, user, first); got != 1.75 {
		t.Fatalf("confident error ease = %v, want 1.75", got)
	}
	if got := ease(t, s, user, second); got != 2.15 {
		t.Fatalf("sure correct ease = %v, want 2.15", got)
	}
	var learner, judged models.CheckTurn
	for _, tr := range done.Turns {
		if tr.ItemID == d.Items[0].ID && tr.Role == "learner" && tr.Move == "answer" {
			learner = tr
		}
		if tr.ItemID == d.Items[0].ID && tr.Outcome != "" {
			judged = tr
		}
	}
	if learner.Confidence != "sure" || !judged.ConfidentError || judged.Outcome != "failed" {
		t.Fatalf("turns: learner %+v judged %+v", learner, judged)
	}
	var flagged bool
	if err := testDB(s).QueryRowContext(ctx, `SELECT COALESCE(json_extract(result, '$.items[0].confidentError'), 0) FROM check_sessions WHERE id = ?`, id).
		Scan(&flagged); err != nil || !flagged {
		t.Fatalf("result line must carry confidentError: %v %v", flagged, err)
	}

	off := false
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{AdaptiveIntervals: &off}); err != nil {
		t.Fatal(err)
	}
	if ease(t, s, user, first) != 2.0 || ease(t, s, user, second) != 2.0 {
		t.Fatal("adaptive off must reset ease to 2.0")
	}
}

func ease(t *testing.T, s *Service, user, objectiveID string) float64 {
	t.Helper()
	var e float64
	if err := testDB(s).QueryRowContext(context.Background(), `SELECT ease FROM user_objective_mastery WHERE user_id = ? AND objective_id = ?`,
		user, objectiveID).Scan(&e); err != nil {
		t.Fatal(err)
	}
	return float64(int(e*100+0.5)) / 100
}
