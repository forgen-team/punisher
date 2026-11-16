package learning

import (
	"testing"
	"time"
)

func TestScaledDays(t *testing.T) {
	cases := []struct {
		days int
		ease float64
		want int
	}{
		{1, 2.0, 1}, {7, 2.0, 7}, {30, 2.0, 30}, {90, 2.0, 90},
		{7, 0, 7},     // neutral
		{7, 3.0, 7},   // out of range = neutral
		{1, 1.3, 1},   // at least a day
		{7, 1.3, 5},   // 4.55 → 5
		{30, 2.5, 38}, // 37.5 → 38
		{90, 1.3, 59}, {90, 2.5, 113},
	}
	for _, c := range cases {
		if got := ScaledDays(c.days, c.ease); got != c.want {
			t.Errorf("ScaledDays(%d, %v) = %d, want %d", c.days, c.ease, got, c.want)
		}
	}
}

func TestUpdateEase(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	old := now.Add(-48 * time.Hour)
	fresh := Mastery{Box: 1, LastCreditAt: &old}
	rose := ApplyEvidence(fresh, OutcomeUnaidedCorrect, now, nil)
	repeat := ApplyEvidence(rose, OutcomeUnaidedCorrect, now.Add(time.Hour), nil) // inside MinCreditGap

	cases := []struct {
		name       string
		ease       float64
		prev, next Mastery
		outcome    string
		confidence string
		want       float64
	}{
		{"box rise sure", 2.0, fresh, rose, OutcomeUnaidedCorrect, ConfidenceSure, 2.1},
		{"box rise unsure", 2.0, fresh, rose, OutcomeUnaidedCorrect, ConfidenceUnsure, 2.05},
		{"box rise unknown keeps the outcome, small rise", 2.0, fresh, rose, OutcomeUnaidedCorrect, ConfidenceUnknown, 2.05},
		{"repeat inside the gap", 2.0, rose, repeat, OutcomeUnaidedCorrect, ConfidenceSure, 2.0},
		{"partial", 2.0, fresh, fresh, OutcomeUnaidedPartial, "", 1.95},
		{"failed", 2.0, fresh, fresh, OutcomeFailed, ConfidenceUnsure, 1.8},
		{"confident error replaces the failure step", 2.0, fresh, fresh, OutcomeFailed, ConfidenceSure, 1.7},
		{"aided keeps", 2.0, fresh, fresh, OutcomeAided, ConfidenceSure, 2.0},
		{"held keeps", 1.5, fresh, fresh, OutcomeHeld, "", 1.5},
		{"clamped low", 1.4, fresh, fresh, OutcomeFailed, ConfidenceSure, 1.3},
		{"clamped high", 2.45, fresh, rose, OutcomeUnaidedCorrect, ConfidenceSure, 2.5},
		{"unset ease starts at default", 0, fresh, fresh, OutcomeFailed, "", 1.8},
	}
	for _, c := range cases {
		if got := UpdateEase(c.ease, c.prev, c.next, c.outcome, c.confidence); got != c.want {
			t.Errorf("%s: UpdateEase = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestEaseScalesTheLadder(t *testing.T) {
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC) // 10:00 KST
	old := now.Add(-48 * time.Hour)
	base := Mastery{Box: 1, LastCreditAt: &old, Best: BestUnaided}
	neutral := ApplyEvidence(base, OutcomeUnaidedCorrect, now, nil) // box 2 → 7 days
	eased := base
	eased.Ease = 1.3
	short := ApplyEvidence(eased, OutcomeUnaidedCorrect, now, nil) // 7 × 0.65 → 5 days
	if d := neutral.NextReviewAt.Sub(KSTDayStart(now)); d != 7*24*time.Hour {
		t.Fatalf("neutral interval = %v, want 7 days", d)
	}
	if d := short.NextReviewAt.Sub(KSTDayStart(now)); d != 5*24*time.Hour {
		t.Fatalf("eased interval = %v, want 5 days", d)
	}

	top := Mastery{Box: MaxBox, LastCreditAt: &old, Best: BestUnaided, Ease: 2.5}
	grad := ApplyEvidence(top, OutcomeUnaidedCorrect, now, nil)
	if d := grad.NextReviewAt.Sub(KSTDayStart(now)); d != 113*24*time.Hour {
		t.Fatalf("graduation interval = %v, want 113 days", d)
	}
	if NextDue(Mastery{GradStep: GradPending, Ease: 1.3}, now, nil).Sub(KSTDayStart(now)) != 59*24*time.Hour {
		t.Fatal("NextDue for a pending graduation must scale too")
	}
}

func TestPullDue(t *testing.T) {
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	far := now.Add(20 * 24 * time.Hour)
	near := now.Add(13 * time.Hour)
	soon := Due(0, now, nil)

	got := PullDue(Mastery{Box: 2, NextReviewAt: &far, Lapses: 1}, now, nil)
	if !got.NextReviewAt.Equal(soon) || got.Box != 2 || got.Lapses != 1 {
		t.Fatalf("pulled = %+v, want next %v with box and lapses kept", got, soon)
	}
	if got := PullDue(Mastery{Box: 2, NextReviewAt: &near}, now, nil); !got.NextReviewAt.Equal(near) {
		t.Fatalf("an earlier review must stay: %v", got.NextReviewAt)
	}
	for name, m := range map[string]Mastery{
		"graduated": {Box: 3, GradStep: GradDone},
		"queued":    {Box: 1, Queued: true},
		"verify":    {Verify: true},
		"no date":   {Box: 1},
	} {
		if got := PullDue(m, now, nil); got.NextReviewAt != nil {
			t.Errorf("%s: got a review date %v", name, got.NextReviewAt)
		}
	}
}

func TestConfidence(t *testing.T) {
	if !ConfidentError(ConfidenceSure, OutcomeFailed) || ConfidentError(ConfidenceSure, OutcomeAided) || ConfidentError(ConfidenceUnsure, OutcomeFailed) {
		t.Fatal("confident error is sure + failed only")
	}
	for _, c := range []string{"", ConfidenceSure, ConfidenceUnsure, ConfidenceUnknown} {
		if !ValidConfidence(c) {
			t.Errorf("%q should be valid", c)
		}
	}
	if ValidConfidence("maybe") {
		t.Fatal("unknown value accepted")
	}
}
