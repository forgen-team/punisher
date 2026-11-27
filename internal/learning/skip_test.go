package learning

import (
	"testing"
	"time"
)

// TestApplySkip: a skip is not evidence — only the review date moves to the nearest one.
func TestApplySkip(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	soon := Due(0, now, nil)
	past, later, sooner := now.Add(-time.Hour), now.AddDate(0, 0, 20), now.Add(6*time.Hour)
	fail := now.AddDate(0, 0, -3)
	m := Mastery{Box: 2, Best: BestUnaided, Lapses: 1, LastFailAt: &fail, NextReviewAt: &past, Ease: 2.2}

	got := ApplySkip(m, now, nil)
	if got.Box != 2 || got.Best != BestUnaided || got.Lapses != 1 || got.LastFailAt != &fail || got.Ease != 2.2 {
		t.Fatalf("skip changed the ladder: %+v", got)
	}
	if got.NextReviewAt == nil || !got.NextReviewAt.Equal(m.due(0, now, nil)) {
		t.Fatalf("overdue review → nearest date, got %v", got.NextReviewAt)
	}
	m.NextReviewAt = &later
	if got := ApplySkip(m, now, nil); !got.NextReviewAt.Equal(m.due(0, now, nil)) {
		t.Fatalf("a far review is pulled to the nearest date, got %v", got.NextReviewAt)
	}
	m.NextReviewAt, m.Ease = &sooner, 0
	if got := ApplySkip(m, now, nil); !got.NextReviewAt.Equal(sooner) || !sooner.Before(soon) {
		t.Fatalf("a sooner review is kept, got %v (soon %v)", got.NextReviewAt, soon)
	}
	if got := ApplySkip(NewMastery(), now, nil); got.NextReviewAt == nil || !got.NextReviewAt.Equal(soon) || got.Best != BestNone {
		t.Fatalf("a new objective enters the schedule: %+v", got)
	}
	for _, keep := range []Mastery{{GradStep: GradDone, Best: BestUnaided}, {Queued: true, Best: BestAided}, {Verify: true, Best: BestNone}} {
		if got := ApplySkip(keep, now, nil); got.NextReviewAt != nil {
			t.Fatalf("graduated / queued / verify keep their state: %+v", got)
		}
	}
}
