package learning

import (
	"testing"
	"time"
)

func kstAt(day string, hour int) time.Time {
	d, _ := time.ParseInLocation("2006-01-02", day, kst)
	return d.Add(time.Duration(hour) * time.Hour)
}

func TestFSRSFirstAndNextStability(t *testing.T) {
	now := kstAt("2026-11-25", 10)
	m := NextMemory(Memory{}, RatingGood, now)
	if m.Reps != 1 || m.Stability != fsrsW[2] || m.Difficulty < 1 || m.Difficulty > 10 {
		t.Fatalf("first good = %+v", m)
	}
	later := now.AddDate(0, 0, FSRSInterval(m.Stability, 0.9))
	good := NextMemory(m, RatingGood, later)
	if good.Stability <= m.Stability {
		t.Fatalf("a good answer at the due date must grow stability: %v → %v", m.Stability, good.Stability)
	}
	again := NextMemory(m, RatingAgain, later)
	if again.Stability > m.Stability || again.Difficulty <= m.Difficulty {
		t.Fatalf("a failure must not grow stability and must raise difficulty: %+v → %+v", m, again)
	}
	if NextMemory(m, 0, later) != m {
		t.Fatal("rating 0 (held) leaves the memory alone")
	}
}

func TestFSRSIntervalFollowsRetention(t *testing.T) {
	if d := FSRSInterval(10, 0.9); d != 10 {
		t.Fatalf("at 90%% retention the interval equals stability, got %d", d)
	}
	if FSRSInterval(10, 0.95) >= FSRSInterval(10, 0.85) {
		t.Fatal("a higher target retention must give shorter intervals")
	}
	if FSRSInterval(10, 2) != 10 {
		t.Fatal("an out-of-range retention falls back to 0.9")
	}
}

func TestBlendStartsFromTheLadder(t *testing.T) {
	if got := BlendDays(7, 20, 0); got != 7 {
		t.Fatalf("no FSRS history keeps the ladder interval: %d", got)
	}
	if got := BlendDays(7, 20, FSRSBlendReps); got != 20 {
		t.Fatalf("enough history uses FSRS alone: %d", got)
	}
	if got := BlendDays(8, 16, FSRSBlendReps/2); got != 12 {
		t.Fatalf("halfway blends: %d", got)
	}
	now := kstAt("2026-11-25", 10)
	ladder := dueIn(7, now)
	if got := RetimeReview(ladder, now, Memory{Stability: 30}, 0, 0.9); !got.Equal(ladder) {
		t.Fatalf("retime with no history = %v, want ladder %v", got, ladder)
	}
}

func TestRatingFor(t *testing.T) {
	cases := map[string]int{OutcomeUnaidedCorrect: RatingGood, OutcomeUnaidedPartial: RatingHard, OutcomeAided: RatingAgain,
		OutcomeFailed: RatingAgain, OutcomeHeld: 0, OutcomeRecorded: 0}
	for outcome, want := range cases {
		if got := RatingFor(outcome); got != want {
			t.Errorf("%s → %d, want %d", outcome, got, want)
		}
	}
}

func TestLeech(t *testing.T) {
	if !CountsAsFailure(OutcomeAided) || CountsAsFailure(OutcomeUnaidedPartial) {
		t.Fatal("aided and failed count, partial does not")
	}
	if BecomesLeech(LeechThreshold-1, false) || !BecomesLeech(LeechThreshold, false) || BecomesLeech(LeechThreshold+1, true) {
		t.Fatal("leech at the threshold, once")
	}
}

func TestRestDays(t *testing.T) {
	if !ValidRestDays([]int{0, 6}) || ValidRestDays([]int{1, 1}) || ValidRestDays([]int{7}) || ValidRestDays([]int{0, 1, 2, 3, 4, 5, 6}) {
		t.Fatal("rest day validation")
	}
	// 2026-11-29 is a Sunday.
	if !IsRestDay(kstAt("2026-11-29", 23), []int{0}) || IsRestDay(kstAt("2026-11-30", 1), []int{0}) {
		t.Fatal("rest day by KST weekday")
	}
}

func TestPlanDeadline(t *testing.T) {
	now := kstAt("2026-11-25", 10) // Wednesday
	p := PlanDeadline(3, "2026-11-27", now, 30, nil)
	if p.StudyDays != 3 || p.RemainingMinutes != 60 || p.MinutesPerDay != 20 || p.Status != PlanOnTrack || p.FinishOn != "2026-11-26" {
		t.Fatalf("on track plan = %+v", p)
	}
	tight := PlanDeadline(6, "2026-11-26", now, 30, nil)
	if tight.Status != PlanTight || tight.MinutesPerDay != 60 {
		t.Fatalf("tight plan = %+v", tight)
	}
	// 토·일 쉬면 금요일까지 3일(수·목·금).
	weekend := PlanDeadline(3, "2026-11-29", now, 30, []int{0, 6})
	if weekend.StudyDays != 3 {
		t.Fatalf("rest days are not study days: %+v", weekend)
	}
	if PlanDeadline(0, "2026-11-27", now, 30, nil).Status != PlanDone || PlanDeadline(2, "", now, 30, nil).Status != PlanNoDeadline {
		t.Fatal("done / no deadline")
	}
	if ValidDueOn("2026-11-24", now) || !ValidDueOn("2026-11-25", now) || ValidDueOn("nope", now) {
		t.Fatal("due date validation")
	}
}
