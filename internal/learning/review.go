package learning

import (
	"sort"
	"time"
)

// Minutes one review item takes by grader (design §2.6 [추정]).
const (
	SelfGradeMinutes  = 1
	TutorGradeMinutes = 2
)

// Inflow control (design §2.6, §9): new checks wait (queued) while the due
// reviews exceed HoldBacklogDays of throughput, and are released once they
// drop below ReleaseBacklogDays.
const (
	HoldBacklogDays    = 3
	ReleaseBacklogDays = 1
)

// MinutesPerItem is the time of one review item for a grader (llm = tutor, anything else = self).
func MinutesPerItem(grader string) int {
	if grader == GraderLLM {
		return TutorGradeMinutes
	}
	return SelfGradeMinutes
}

// ReviewThroughput is the review items per day: review_minutes ÷ minutes per item (0 = reviews off).
func ReviewThroughput(reviewMinutes int, grader string) int {
	per := MinutesPerItem(grader)
	if per <= 0 || reviewMinutes <= 0 {
		return 0
	}
	return reviewMinutes / per
}

// HoldNewReviews: the backlog exceeds HoldBacklogDays of throughput, so
// objectives entering from a new check wait (queued). No throughput (reviews
// off) gives no limit to measure against: nothing is held.
func HoldNewReviews(backlog, throughput int) bool {
	return throughput > 0 && backlog > HoldBacklogDays*throughput
}

// ReleaseCount is how many queued objectives enter the queue now: once the
// backlog is below ReleaseBacklogDays of throughput, up to that level.
func ReleaseCount(backlog, throughput int) int {
	limit := ReleaseBacklogDays * throughput
	if throughput <= 0 || backlog >= limit {
		return 0
	}
	return limit - backlog
}

// QueueEntry applies inflow control to evidence just applied (prev → next).
// An objective already waiting keeps waiting; one entering the review
// schedule for the first time (no review date before) waits when hold is set.
// A waiting objective has no review date until it is released.
func QueueEntry(prev, next Mastery, hold bool) Mastery {
	if next.NextReviewAt == nil {
		next.Queued = false
		return next
	}
	if prev.Queued || (hold && prev.NextReviewAt == nil) {
		next.Queued, next.NextReviewAt = true, nil
	}
	return next
}

// NextDue is the review date given to a released objective: the graduation
// review for a pending graduation, else Due(box).
func NextDue(m Mastery, now time.Time, intervals []int) time.Time {
	if m.GradStep == GradPending {
		return dueIn(GraduationDays, now)
	}
	return Due(m.Box, now, intervals)
}

// ReviewInterval is the interval (days) a review at this state tests: the
// graduation review is GraduationDays, otherwise intervals[clamp(box-1)] (as Due).
func ReviewInterval(box, gradStep int, intervals []int) int {
	if gradStep == GradPending {
		return GraduationDays
	}
	if len(intervals) == 0 {
		intervals = DefaultReviewIntervals
	}
	return intervals[min(max(box-1, 0), len(intervals)-1)]
}

// ReviewCandidate is a due objective of the review queue.
type ReviewCandidate struct {
	ObjectiveID  string
	ConceptID    string
	Area         string
	Box          int
	GradStep     int
	NextReviewAt time.Time
}

// ReviewScore = overdue days / interval + 1/(box+1) (design §2.6): low boxes
// first, and a high box rises as its delay grows relative to its interval.
func ReviewScore(c ReviewCandidate, now time.Time, intervals []int) float64 {
	overdue := now.Sub(c.NextReviewAt).Hours() / 24
	if overdue < 0 {
		overdue = 0
	}
	interval := ReviewInterval(c.Box, c.GradStep, intervals)
	score := 1 / float64(max(c.Box, 0)+1)
	if interval > 0 {
		score += overdue / float64(interval)
	}
	return score
}

// OrderReviews sorts the queue by score (ties: earlier due, then id), then
// spreads areas so the same area never comes twice in a row when avoidable.
func OrderReviews(list []ReviewCandidate, now time.Time, intervals []int) []ReviewCandidate {
	out := append([]ReviewCandidate{}, list...)
	scores := make(map[string]float64, len(out))
	for _, c := range out {
		scores[c.ObjectiveID] = ReviewScore(c, now, intervals)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if scores[a.ObjectiveID] != scores[b.ObjectiveID] {
			return scores[a.ObjectiveID] > scores[b.ObjectiveID]
		}
		if !a.NextReviewAt.Equal(b.NextReviewAt) {
			return a.NextReviewAt.Before(b.NextReviewAt)
		}
		return a.ObjectiveID < b.ObjectiveID
	})
	return SpreadAreas(out)
}

// SpreadAreas reorders a ranked list greedily: each next item is the
// highest-ranked one whose area differs from the previous item's; when only
// the same area is left, the order is kept.
func SpreadAreas(ranked []ReviewCandidate) []ReviewCandidate {
	rest := append([]ReviewCandidate{}, ranked...)
	out := make([]ReviewCandidate, 0, len(rest))
	for len(rest) > 0 {
		pick := 0
		if len(out) > 0 {
			prev := out[len(out)-1].Area
			for i, c := range rest {
				if c.Area != prev {
					pick = i
					break
				}
			}
		}
		out = append(out, rest[pick])
		rest = append(rest[:pick], rest[pick+1:]...)
	}
	return out
}

// ReviewCounts are the user's review queue numbers at a moment.
type ReviewCounts struct {
	// Due: next_review_at ≤ now (in the queue). Overdue: due before today (KST).
	Due     int
	Overdue int
	// Queued: waiting by inflow control (no review date yet).
	Queued int
}

// ReviewToday is the review part of the today payload (TodayResult.Review).
type ReviewToday struct {
	Due              int `json:"due"`
	Overdue          int `json:"overdue"`
	Queued           int `json:"queued"`
	EstimatedMinutes int `json:"estimatedMinutes"`
	// Backlogged: due reviews exceed HoldBacklogDays of throughput ("밀린 복습 n개" · '오늘 더 하기').
	Backlogged     bool `json:"backlogged"`
	LectureMinutes int  `json:"lectureMinutes"`
}

// SummarizeReviews turns counts and the settings into today's review numbers:
// the estimate covers what fits in today's session (min(due, throughput) items).
func SummarizeReviews(c ReviewCounts, reviewMinutes int, grader string) ReviewToday {
	throughput := ReviewThroughput(reviewMinutes, grader)
	return ReviewToday{Due: c.Due, Overdue: c.Overdue, Queued: c.Queued,
		EstimatedMinutes: min(c.Due, throughput) * MinutesPerItem(grader),
		Backlogged:       HoldNewReviews(c.Due, throughput), LectureMinutes: DefaultLectureMinutes}
}

// RetentionSample is one first attempt of a review item on an unseen item.
type RetentionSample struct {
	Grader string
	// PrevBox / PrevGradStep: the state before the review (which interval it tested).
	PrevBox      int
	PrevGradStep int
	Correct      bool
}

// RetentionBucket is the first-attempt accuracy of one interval and grader.
type RetentionBucket struct {
	IntervalDays int    `json:"intervalDays"`
	Grader       string `json:"grader"`
	Attempts     int    `json:"attempts"`
	Correct      int    `json:"correct"`
	// Rate is nil when there is no attempt (no 0/0).
	Rate *float64 `json:"rate"`
}

// RetentionMetrics groups samples by tested interval (the ladder + graduation)
// and grader (design §2.7). Every interval × {llm, self} bucket is listed.
func RetentionMetrics(samples []RetentionSample, intervals []int) []RetentionBucket {
	if len(intervals) == 0 {
		intervals = DefaultReviewIntervals
	}
	days := append(append([]int{}, intervals...), GraduationDays)
	out := []RetentionBucket{}
	index := map[[2]any]int{}
	for _, d := range days {
		for _, g := range []string{GraderLLM, GraderSelf} {
			key := [2]any{d, g}
			if _, ok := index[key]; ok {
				continue
			}
			index[key] = len(out)
			out = append(out, RetentionBucket{IntervalDays: d, Grader: g})
		}
	}
	for _, s := range samples {
		i, ok := index[[2]any{ReviewInterval(s.PrevBox, s.PrevGradStep, intervals), s.Grader}]
		if !ok {
			continue
		}
		out[i].Attempts++
		if s.Correct {
			out[i].Correct++
		}
	}
	for i := range out {
		if out[i].Attempts > 0 {
			rate := float64(out[i].Correct) / float64(out[i].Attempts)
			out[i].Rate = &rate
		}
	}
	return out
}
