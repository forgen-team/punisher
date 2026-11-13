package learning

import (
	"strings"
	"testing"
	"time"
)

func TestThroughput(t *testing.T) {
	cases := []struct {
		minutes int
		grader  string
		want    int
	}{
		{10, GraderSelf, 10}, {10, GraderLLM, 5}, {11, GraderLLM, 5}, {0, GraderSelf, 0}, {-3, GraderSelf, 0}, {30, "", 30},
	}
	for _, c := range cases {
		if got := ReviewThroughput(c.minutes, c.grader); got != c.want {
			t.Errorf("ReviewThroughput(%d,%q) = %d, want %d", c.minutes, c.grader, got, c.want)
		}
	}
}

func TestInflowControl(t *testing.T) {
	cases := []struct {
		name             string
		backlog, through int
		hold             bool
		release          int
	}{
		{"exactly 3 days → no hold", 30, 10, false, 0},
		{"over 3 days → hold", 31, 10, true, 0},
		{"1 day → no release", 10, 10, false, 0},
		{"under 1 day → release up to 1 day", 7, 10, false, 3},
		{"empty → release a day", 0, 10, false, 10},
		{"reviews off (0) → no hold, no release (no 0 division)", 500, 0, false, 0},
	}
	for _, c := range cases {
		if got := HoldNewReviews(c.backlog, c.through); got != c.hold {
			t.Errorf("%s: hold = %v", c.name, got)
		}
		if got := ReleaseCount(c.backlog, c.through); got != c.release {
			t.Errorf("%s: release = %d", c.name, got)
		}
	}
}

func TestQueueEntry(t *testing.T) {
	next := Mastery{Box: 1, Best: BestUnaided, NextReviewAt: at(24 * time.Hour)}
	if m := QueueEntry(NewMastery(), next, true); !m.Queued || m.NextReviewAt != nil {
		t.Fatalf("a new objective under hold waits: %+v", m)
	}
	if m := QueueEntry(NewMastery(), next, false); m.Queued || m.NextReviewAt == nil {
		t.Fatalf("no hold → scheduled: %+v", m)
	}
	scheduled := Mastery{Box: 1, NextReviewAt: at(time.Hour)}
	if m := QueueEntry(scheduled, next, true); m.Queued || m.NextReviewAt == nil {
		t.Fatalf("an objective already in the schedule is not held: %+v", m)
	}
	waiting := Mastery{Box: 1, Queued: true}
	if m := QueueEntry(waiting, next, false); !m.Queued || m.NextReviewAt != nil {
		t.Fatalf("a waiting objective keeps waiting until released: %+v", m)
	}
	done := Mastery{Box: 3, GradStep: GradDone}
	if m := QueueEntry(waiting, done, true); m.Queued {
		t.Fatalf("no review date → not queued: %+v", m)
	}
}

func TestNextDueAndInterval(t *testing.T) {
	if got := NextDue(Mastery{Box: 2}, masteryNow, nil); !got.Equal(kstDay(7)) {
		t.Fatalf("released box 2 → 7 days: %v", got)
	}
	if got := NextDue(Mastery{Box: 3, GradStep: GradPending}, masteryNow, nil); !got.Equal(kstDay(GraduationDays)) {
		t.Fatalf("pending graduation → 90 days: %v", got)
	}
	for _, c := range []struct{ box, grad, want int }{{0, 0, 1}, {1, 0, 1}, {2, 0, 7}, {3, 0, 30}, {9, 0, 30}, {3, GradPending, 90}} {
		if got := ReviewInterval(c.box, c.grad, nil); got != c.want {
			t.Errorf("ReviewInterval(%d,%d) = %d, want %d", c.box, c.grad, got, c.want)
		}
	}
}

func cand(id, area string, box int, dueAgo time.Duration) ReviewCandidate {
	return ReviewCandidate{ObjectiveID: id, Area: area, Box: box, NextReviewAt: masteryNow.Add(-dueAgo)}
}

func TestReviewScore(t *testing.T) {
	day := 24 * time.Hour
	if got := ReviewScore(cand("a", "x", 0, 0), masteryNow, nil); got != 1 {
		t.Fatalf("box 0 due now = %v", got)
	}
	if got := ReviewScore(cand("a", "x", 2, 14*day), masteryNow, nil); got != 2+1.0/3 {
		t.Fatalf("box 2 two intervals late = %v", got)
	}
	if got := ReviewScore(cand("a", "x", 1, -day), masteryNow, nil); got != 0.5 {
		t.Fatalf("not yet due counts no delay = %v", got)
	}
}

func ids(list []ReviewCandidate) string {
	out := make([]string, 0, len(list))
	for _, c := range list {
		out = append(out, c.ObjectiveID)
	}
	return strings.Join(out, ",")
}

func TestOrderReviews(t *testing.T) {
	day := 24 * time.Hour
	queue := []ReviewCandidate{
		cand("high-box-fresh", "os", 3, 0),      // 0.25
		cand("low-box", "os", 0, 0),             // 1
		cand("high-box-late", "net", 3, 60*day), // 2.25: 연체가 길면 위로
		cand("mid", "net", 1, 2*day),            // 2.5
	}
	if got := ids(OrderReviews(queue, masteryNow, nil)); got != "mid,low-box,high-box-late,high-box-fresh" {
		t.Fatalf("order = %s", got)
	}
	tie := []ReviewCandidate{cand("b", "os", 0, 0), cand("a", "os", 0, 0), cand("c", "os", 0, time.Hour)}
	if got := ids(OrderReviews(tie, masteryNow, nil)); got != "c,a,b" {
		t.Fatalf("ties: earlier due, then id = %s", got)
	}
}

func TestSpreadAreas(t *testing.T) {
	ranked := []ReviewCandidate{cand("o1", "os", 0, 0), cand("o2", "os", 0, 0), cand("n1", "net", 0, 0),
		cand("o3", "os", 0, 0), cand("d1", "db", 0, 0)}
	if got := ids(SpreadAreas(ranked)); got != "o1,n1,o2,d1,o3" {
		t.Fatalf("spread = %s", got)
	}
	same := []ReviewCandidate{cand("o1", "os", 0, 0), cand("o2", "os", 0, 0)}
	if got := ids(SpreadAreas(same)); got != "o1,o2" {
		t.Fatalf("only one area left keeps the order = %s", got)
	}
}

func TestSummarizeReviews(t *testing.T) {
	r := SummarizeReviews(ReviewCounts{Due: 31, Overdue: 20, Queued: 4}, 10, GraderSelf)
	if r.EstimatedMinutes != 10 || !r.Backlogged || r.LectureMinutes != DefaultLectureMinutes || r.Queued != 4 {
		t.Fatalf("backlogged summary = %+v", r)
	}
	r = SummarizeReviews(ReviewCounts{Due: 3}, 10, GraderLLM)
	if r.EstimatedMinutes != 6 || r.Backlogged {
		t.Fatalf("tutor summary = %+v", r)
	}
	if r = SummarizeReviews(ReviewCounts{Due: 3}, 0, GraderSelf); r.EstimatedMinutes != 0 || r.Backlogged {
		t.Fatalf("reviews off = %+v", r)
	}
}

func TestRetentionMetrics(t *testing.T) {
	samples := []RetentionSample{
		{Grader: GraderSelf, PrevBox: 1, Correct: true},
		{Grader: GraderSelf, PrevBox: 0, Correct: false},
		{Grader: GraderLLM, PrevBox: 2, Correct: true},
		{Grader: GraderSelf, PrevBox: 3, PrevGradStep: GradPending, Correct: true},
	}
	got := RetentionMetrics(samples, nil)
	if len(got) != 8 {
		t.Fatalf("buckets = %d, want 4 intervals × 2 graders", len(got))
	}
	find := func(days int, grader string) RetentionBucket {
		for _, b := range got {
			if b.IntervalDays == days && b.Grader == grader {
				return b
			}
		}
		t.Fatalf("bucket %d/%s missing", days, grader)
		return RetentionBucket{}
	}
	if b := find(1, GraderSelf); b.Attempts != 2 || b.Correct != 1 || b.Rate == nil || *b.Rate != 0.5 {
		t.Fatalf("1 day self = %+v", b)
	}
	if b := find(7, GraderLLM); b.Attempts != 1 || *b.Rate != 1 {
		t.Fatalf("7 day llm = %+v", b)
	}
	if b := find(90, GraderSelf); b.Attempts != 1 {
		t.Fatalf("graduation = %+v", b)
	}
	if b := find(30, GraderSelf); b.Attempts != 0 || b.Rate != nil {
		t.Fatalf("empty bucket rate must be null: %+v", b)
	}
}

func TestBuildTodayReviewField(t *testing.T) {
	res := BuildToday(TodayInput{Review: ReviewInput{Counts: ReviewCounts{Due: 40, Overdue: 30}, ReviewMinutes: 10, Grader: GraderSelf}})
	if !res.Exhausted || !res.Review.Backlogged || res.Review.Due != 40 || res.Review.EstimatedMinutes != 10 {
		t.Fatalf("review field must not change Exhausted: %+v", res)
	}
}
