package learning

import (
	"fmt"
	"testing"
	"time"
)

// now: 2026-11-13 15:00 KST.
var masteryNow = time.Date(2026, 11, 13, 15, 0, 0, 0, kst)

func at(d time.Duration) *time.Time { t := masteryNow.Add(d); return &t }

func kstDay(days int) time.Time { return KSTDayStart(masteryNow).AddDate(0, 0, days) }

func TestDue(t *testing.T) {
	late := time.Date(2026, 11, 13, 23, 30, 0, 0, kst) // 자정 직전: 다음 날 0시는 12시간 안
	early := time.Date(2026, 11, 13, 0, 10, 0, 0, kst) // 자정 직후
	cases := []struct {
		name      string
		box       int
		now       time.Time
		intervals []int
		want      time.Time
	}{
		{"box 0 at 15:00 → 12h floor", 0, masteryNow, nil, masteryNow.Add(12 * time.Hour)},
		{"box 1 at 09:00 → next KST day", 1, masteryNow.Add(-6 * time.Hour), []int{1, 7, 30}, kstDay(1)},
		{"box 2 → 7 days", 2, masteryNow, []int{1, 7, 30}, kstDay(7)},
		{"box 3 → 30 days", 3, masteryNow, []int{1, 7, 30}, kstDay(30)},
		{"box beyond ladder → last interval", 5, masteryNow, []int{1, 7}, kstDay(7)},
		{"late night: at least 12h", 1, late, []int{1, 7, 30}, late.Add(12 * time.Hour)},
		{"just after midnight: next KST day", 1, early, []int{1, 7, 30}, time.Date(2026, 11, 14, 0, 0, 0, 0, kst)},
		{"empty intervals use defaults", 2, masteryNow, []int{}, kstDay(7)},
	}
	for _, c := range cases {
		if got := Due(c.box, c.now, c.intervals); !got.Equal(c.want) {
			t.Errorf("%s: Due = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestItemOutcome(t *testing.T) {
	cases := []struct {
		name     string
		r        ItemResult
		lastFail *time.Time
		want     string
	}{
		{"first answer correct", ItemResult{FirstVerdict: VerdictCorrect, FinalVerdict: VerdictCorrect}, nil, OutcomeUnaidedCorrect},
		{"correct, failure 13h ago", ItemResult{FirstVerdict: VerdictCorrect, FinalVerdict: VerdictCorrect}, at(-13 * time.Hour), OutcomeUnaidedCorrect},
		{"correct exactly 12h after failure", ItemResult{FirstVerdict: VerdictCorrect, FinalVerdict: VerdictCorrect}, at(-12 * time.Hour), OutcomeUnaidedCorrect},
		{"correct within 12h of a failure → aided", ItemResult{FirstVerdict: VerdictCorrect, FinalVerdict: VerdictCorrect}, at(-11 * time.Hour), OutcomeAided},
		{"first partial, item ended", ItemResult{FirstVerdict: VerdictPartial, FinalVerdict: VerdictPartial}, nil, OutcomeUnaidedPartial},
		{"partial within 12h of a failure → aided", ItemResult{FirstVerdict: VerdictPartial, FinalVerdict: VerdictPartial}, at(-time.Hour), OutcomeAided},
		{"correct after narrowing", ItemResult{FirstVerdict: VerdictPartial, FinalVerdict: VerdictCorrect, Aided: true}, nil, OutcomeAided},
		{"wrong after narrowing", ItemResult{FirstVerdict: VerdictIncorrect, FinalVerdict: VerdictIncorrect, Aided: true}, nil, OutcomeFailed},
		{"partial after explanation", ItemResult{FirstVerdict: VerdictIncorrect, FinalVerdict: VerdictPartial, Aided: true}, nil, OutcomeFailed},
		{"first answer wrong (left)", ItemResult{FirstVerdict: VerdictIncorrect, FinalVerdict: VerdictIncorrect}, nil, OutcomeFailed},
		{"revealed", ItemResult{Revealed: true}, nil, OutcomeFailed},
		{"revealed after a partial", ItemResult{FirstVerdict: VerdictPartial, FinalVerdict: VerdictPartial, Revealed: true}, nil, OutcomeFailed},
		{"held first answer", ItemResult{FirstVerdict: VerdictHeld, FinalVerdict: VerdictHeld}, nil, OutcomeHeld},
		{"held after narrowing", ItemResult{FirstVerdict: VerdictPartial, FinalVerdict: VerdictHeld, Aided: true}, nil, OutcomeHeld},
	}
	for _, c := range cases {
		if got := ItemOutcome(c.r, c.lastFail, masteryNow); got != c.want {
			t.Errorf("%s: outcome = %s, want %s", c.name, got, c.want)
		}
	}
}

func TestApplyEvidenceTable(t *testing.T) {
	iv := []int{1, 7, 30}
	next := func(days int) *time.Time { d := kstDay(days); return &d }
	keepNext := at(72 * time.Hour)
	cases := []struct {
		name    string
		in      Mastery
		outcome string
		want    Mastery
	}{
		{"unaided_correct, never credited → box+1", NewMastery(), OutcomeUnaidedCorrect,
			Mastery{Box: 1, Best: BestUnaided, NextReviewAt: at(12 * time.Hour), LastCreditAt: at(0)}},
		{"unaided_correct, credited 12h+ ago → box+1", Mastery{Box: 1, Best: BestUnaided, LastCreditAt: at(-13 * time.Hour), NextReviewAt: keepNext}, OutcomeUnaidedCorrect,
			Mastery{Box: 2, Best: BestUnaided, NextReviewAt: next(7), LastCreditAt: at(0)}},
		{"unaided_correct, credited <12h ago → box max(box,1), next kept", Mastery{Box: 0, Best: BestAided, LastCreditAt: at(-time.Hour), NextReviewAt: keepNext}, OutcomeUnaidedCorrect,
			Mastery{Box: 1, Best: BestUnaided, LastCreditAt: at(-time.Hour), NextReviewAt: keepNext}},
		{"unaided_correct, <12h, box 2 stays", Mastery{Box: 2, Best: BestUnaided, LastCreditAt: at(-time.Hour), NextReviewAt: keepNext}, OutcomeUnaidedCorrect,
			Mastery{Box: 2, Best: BestUnaided, LastCreditAt: at(-time.Hour), NextReviewAt: keepNext}},
		{"unaided_correct, <12h, no next yet (after relearn) → due(box)", Mastery{Box: 0, Best: BestUnaided, LastCreditAt: at(-time.Hour)}, OutcomeUnaidedCorrect,
			Mastery{Box: 1, Best: BestUnaided, LastCreditAt: at(-time.Hour), NextReviewAt: at(12 * time.Hour)}},
		{"unaided_correct, box 2 → 3 (30 days)", Mastery{Box: 2, Best: BestUnaided, LastCreditAt: at(-8 * 24 * time.Hour)}, OutcomeUnaidedCorrect,
			Mastery{Box: 3, Best: BestUnaided, NextReviewAt: next(30), LastCreditAt: at(0)}},
		{"unaided_correct at box 3 → graduation review in 90 days", Mastery{Box: 3, Best: BestUnaided, LastCreditAt: at(-31 * 24 * time.Hour)}, OutcomeUnaidedCorrect,
			Mastery{Box: 3, GradStep: GradPending, Best: BestUnaided, NextReviewAt: next(GraduationDays), LastCreditAt: at(0)}},
		{"unaided_correct on the graduation review → done, leaves the queue", Mastery{Box: 3, GradStep: GradPending, Best: BestUnaided, LastCreditAt: at(-91 * 24 * time.Hour), NextReviewAt: keepNext}, OutcomeUnaidedCorrect,
			Mastery{Box: 3, GradStep: GradDone, Best: BestUnaided, LastCreditAt: at(0)}},
		{"recovery: box back to 2 resets lapses", Mastery{Box: 1, Best: BestUnaided, Lapses: 2, LastCreditAt: at(-24 * time.Hour)}, OutcomeUnaidedCorrect,
			Mastery{Box: 2, Best: BestUnaided, NextReviewAt: next(7), LastCreditAt: at(0)}},
		{"box 0 → 1 keeps lapses (not recovered)", Mastery{Box: 0, Best: BestUnaided, Lapses: 2, LastCreditAt: at(-24 * time.Hour)}, OutcomeUnaidedCorrect,
			Mastery{Box: 1, Best: BestUnaided, Lapses: 2, NextReviewAt: at(12 * time.Hour), LastCreditAt: at(0)}},
		{"unaided_partial: box kept, next = due(box-1)", Mastery{Box: 3, Best: BestUnaided, NextReviewAt: keepNext}, OutcomeUnaidedPartial,
			Mastery{Box: 3, Best: BestUnaided, NextReviewAt: next(7)}},
		{"unaided_partial at box 0 → due(0)", NewMastery(), OutcomeUnaidedPartial,
			Mastery{Box: 0, Best: BestNone, NextReviewAt: at(12 * time.Hour)}},
		{"aided from box 2 → box 0, lapse", Mastery{Box: 2, Best: BestUnaided, LastCreditAt: at(-time.Hour)}, OutcomeAided,
			Mastery{Box: 0, Best: BestUnaided, Lapses: 1, LastFailAt: at(0), LastCreditAt: at(-time.Hour), NextReviewAt: at(12 * time.Hour)}},
		{"aided from box 1 → no lapse, best raised", Mastery{Box: 1, Best: BestRevealed}, OutcomeAided,
			Mastery{Box: 0, Best: BestAided, LastFailAt: at(0), NextReviewAt: at(12 * time.Hour)}},
		{"failed from box 3 in graduation → ladder, lapse", Mastery{Box: 3, GradStep: GradPending, Best: BestUnaided, IntegrativeOK: true}, OutcomeFailed,
			Mastery{Box: 0, GradStep: GradLadder, Best: BestUnaided, IntegrativeOK: true, Lapses: 1, LastFailAt: at(0), NextReviewAt: at(12 * time.Hour)}},
		{"failed first time → best revealed", NewMastery(), OutcomeFailed,
			Mastery{Box: 0, Best: BestRevealed, LastFailAt: at(0), NextReviewAt: at(12 * time.Hour)}},
		{"held: no change (stale kept)", Mastery{Box: 2, Best: BestUnaided, Stale: true, NextReviewAt: keepNext}, OutcomeHeld,
			Mastery{Box: 2, Best: BestUnaided, Stale: true, NextReviewAt: keepNext}},
		{"new evidence clears stale", Mastery{Box: 1, Best: BestUnaided, Stale: true}, OutcomeUnaidedCorrect,
			Mastery{Box: 2, Best: BestUnaided, NextReviewAt: next(7), LastCreditAt: at(0)}},
	}
	for _, c := range cases {
		got := ApplyEvidence(c.in, c.outcome, masteryNow, iv)
		if !sameMastery(got, c.want) {
			t.Errorf("%s:\n got  %s\n want %s", c.name, fmtMastery(got), fmtMastery(c.want))
		}
	}
}

func TestRestore(t *testing.T) {
	prev := Mastery{Box: 2, Best: BestAided, Lapses: 1, NextReviewAt: at(time.Hour), LastCreditAt: at(-48 * time.Hour)}
	applied := ApplyEvidence(prev, OutcomeFailed, masteryNow, nil)
	got := Restore(applied, prev, false)
	want := prev
	want.Best = BestUnaided
	if !sameMastery(got, want) {
		t.Fatalf("restore: %s, want %s (box not raised, best unaided)", fmtMastery(got), fmtMastery(want))
	}
	other := ApplyEvidence(applied, OutcomeUnaidedCorrect, masteryNow.Add(time.Hour), nil)
	got = Restore(other, prev, true)
	want = other
	want.Best = BestUnaided
	if !sameMastery(got, want) {
		t.Fatalf("conflict keeps current, raises best only: %s", fmtMastery(got))
	}
}

func TestMarkStale(t *testing.T) {
	got := MarkStale(Mastery{Box: 3, GradStep: GradPending, Best: BestUnaided, IntegrativeOK: true, Lapses: 1})
	want := Mastery{Box: 1, GradStep: GradLadder, Best: BestUnaided, Lapses: 1, Stale: true}
	if !sameMastery(got, want) {
		t.Fatalf("stale: %s", fmtMastery(got))
	}
	if DisplayLevel(got) != LevelAlone {
		t.Fatal("a stale objective drops to level 3 at most")
	}
	if low := MarkStale(Mastery{Box: 0, Best: BestAided}); low.Box != 0 || !low.Stale {
		t.Fatalf("box below 1 is kept: %+v", low)
	}
}

func TestDisplayAndConceptLevel(t *testing.T) {
	cases := []struct {
		m    Mastery
		want int
	}{
		{NewMastery(), LevelUnchecked},
		{Mastery{Best: BestRevealed}, LevelHeard},
		{Mastery{Best: BestAided}, LevelFollowed},
		{Mastery{Best: BestUnaided, Box: 0}, LevelAlone},
		{Mastery{Best: BestUnaided, Box: 1}, LevelAlone},
		{Mastery{Best: BestUnaided, Box: 2}, LevelRemember},
		{Mastery{Best: BestUnaided, Box: 3}, LevelRemember},
		{Mastery{Best: BestUnaided, Box: 2, IntegrativeOK: true}, LevelRemember},
		{Mastery{Best: BestUnaided, Box: 3, IntegrativeOK: true}, LevelExplain},
	}
	for _, c := range cases {
		if got := DisplayLevel(c.m); got != c.want {
			t.Errorf("DisplayLevel(%+v) = %d, want %d", c.m, got, c.want)
		}
	}
	if ConceptLevel(nil) != 0 || ConceptLevel([]int{4, 2, 5}) != 2 {
		t.Fatal("concept level is the minimum")
	}
	if ConceptMastered(nil, 4) || ConceptMastered([]int{4, 3}, 4) || !ConceptMastered([]int{4, 5}, 4) {
		t.Fatal("mastered = every objective at the target")
	}
	if MaxBest(BestAided, BestRevealed) != BestAided || MaxBest(BestNone, BestUnaided) != BestUnaided {
		t.Fatal("best order")
	}
}

func TestMasterySettingsValidation(t *testing.T) {
	for _, c := range []struct {
		days []int
		ok   bool
	}{
		{[]int{1, 7, 30}, true}, {[]int{2}, true}, {[]int{1, 2, 3, 4, 5, 6}, true},
		{nil, false}, {[]int{}, false}, {[]int{1, 2, 3, 4, 5, 6, 7}, false},
		{[]int{0, 7}, false}, {[]int{7, 7}, false}, {[]int{7, 3}, false},
	} {
		if ValidReviewIntervals(c.days) != c.ok {
			t.Errorf("intervals %v valid=%v", c.days, !c.ok)
		}
	}
	if !ValidMasteryTarget(3) || !ValidMasteryTarget(5) || ValidMasteryTarget(2) || ValidMasteryTarget(6) {
		t.Fatal("mastery target 3~5")
	}
	if !ValidReviewMinutes(0) || !ValidReviewMinutes(30) || ValidReviewMinutes(31) || ValidReviewMinutes(-1) {
		t.Fatal("review minutes 0~30")
	}
	if !ValidReviewGrader("self") || !ValidReviewGrader("llm") || ValidReviewGrader("llm_regrade") {
		t.Fatal("review grader self|llm")
	}
}

func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func sameMastery(a, b Mastery) bool {
	return a.Box == b.Box && a.GradStep == b.GradStep && a.Best == b.Best && a.IntegrativeOK == b.IntegrativeOK &&
		a.Lapses == b.Lapses && a.Stale == b.Stale && sameTime(a.NextReviewAt, b.NextReviewAt) &&
		sameTime(a.LastCreditAt, b.LastCreditAt) && sameTime(a.LastFailAt, b.LastFailAt)
}

func fmtMastery(m Mastery) string {
	f := func(t *time.Time) string {
		if t == nil {
			return "nil"
		}
		return t.In(kst).Format("01-02 15:04")
	}
	return fmt.Sprintf("box=%d grad=%d best=%s lapses=%d next=%s credit=%s fail=%s integ=%v stale=%v",
		m.Box, m.GradStep, m.Best, m.Lapses, f(m.NextReviewAt), f(m.LastCreditAt), f(m.LastFailAt), m.IntegrativeOK, m.Stale)
}

// TestKeepSince: an accepted dispute restores the before-state but keeps what
// changed since without being evidence (release, integrative credit, stale mark).
func TestKeepSince(t *testing.T) {
	now := time.Date(2026, 11, 13, 9, 0, 0, 0, time.UTC)
	due := now.Add(48 * time.Hour)
	waiting := Mastery{Box: 1, Best: BestAided, Queued: true}
	released := Mastery{Box: 0, Best: BestRevealed, NextReviewAt: &due, IntegrativeOK: true}
	got := KeepSince(waiting, released, now, nil)
	if got.Queued || got.NextReviewAt == nil || !got.IntegrativeOK || got.Box != 1 {
		t.Fatalf("released since: %+v (must not wait again, gets a date, keeps the credit)", got)
	}
	stillWaiting := Mastery{Box: 0, Queued: true}
	if got := KeepSince(waiting, stillWaiting, now, nil); !got.Queued || got.NextReviewAt != nil {
		t.Fatalf("still waiting: %+v", got)
	}
	if got := KeepSince(Mastery{Box: 3, IntegrativeOK: true}, Mastery{Box: 1, Stale: true}, now, nil); !got.Stale || got.Box != 1 || got.IntegrativeOK {
		t.Fatalf("stale since: %+v (MarkStale applies)", got)
	}
}
