package learning

import "time"

// Evidence best levels, in order none < revealed < aided < unaided (design §2.3).
const (
	BestNone     = "none"
	BestRevealed = "revealed"
	BestAided    = "aided"
	BestUnaided  = "unaided"
)

// Item outcomes: one per finished check item (design §2.3).
const (
	OutcomeUnaidedCorrect = "unaided_correct"
	OutcomeUnaidedPartial = "unaided_partial"
	OutcomeAided          = "aided"
	OutcomeFailed         = "failed"
	OutcomeHeld           = "held"
	// OutcomeRecorded: a pretest / placement answer — recorded, never evidence.
	OutcomeRecorded = "recorded"
)

// Verdicts of one answer (LLM tutor, regrade or self-grade).
const (
	VerdictCorrect   = "correct"
	VerdictPartial   = "partial"
	VerdictIncorrect = "incorrect"
	VerdictRevealed  = "revealed"
	VerdictHeld      = "held"
)

const (
	// MinCreditGap: an unaided correct answer raises the box only this long after the last raise,
	// and an answer this soon after a failure on the same objective counts as aided.
	MinCreditGap = 12 * time.Hour
	// GraduationDays: after box 3 one more review this many days later, then the objective leaves the queue.
	GraduationDays = 90
	// MaxBox is the top of the ladder; further credit moves to the graduation steps.
	MaxBox = 3
)

// Graduation steps (user_objective_mastery.grad_step).
const (
	GradLadder  = 0 // on the review ladder
	GradPending = 1 // the 90-day graduation review is scheduled
	GradDone    = 2 // graduated: no more reviews, level kept
)

// Display levels (design §2.2).
const (
	LevelUnchecked = 0 // 미확인
	LevelHeard     = 1 // 들어봄
	LevelFollowed  = 2 // 따라감
	LevelAlone     = 3 // 혼자 답함
	LevelRemember  = 4 // 기억함
	LevelExplain   = 5 // 설명 가능
)

// Mastery is one objective's spaced-repetition state (user_objective_mastery),
// also stored as the "before" snapshot (grade.prev) so a dispute can restore it.
type Mastery struct {
	Box           int        `json:"box"`
	GradStep      int        `json:"gradStep"`
	Best          string     `json:"best"`
	IntegrativeOK bool       `json:"integrativeOk"`
	Lapses        int        `json:"lapses"`
	NextReviewAt  *time.Time `json:"nextReviewAt"`
	LastCreditAt  *time.Time `json:"lastCreditAt"`
	LastFailAt    *time.Time `json:"lastFailAt"`
	Stale         bool       `json:"stale"`
	// Queued: waiting by inflow control (design §2.6) — no review date until released.
	Queued bool `json:"queued"`
	// Verify: in the known-verification queue (bulk '이미 앎', design §5 P3).
	Verify bool `json:"verify,omitempty"`
}

// NewMastery is the state of an objective without evidence.
func NewMastery() Mastery { return Mastery{Best: BestNone} }

var bestRank = map[string]int{BestNone: 0, BestRevealed: 1, BestAided: 2, BestUnaided: 3}

// MaxBest returns the higher of two best levels (unknown values rank as none).
func MaxBest(a, b string) string {
	if bestRank[b] > bestRank[a] {
		return b
	}
	if _, ok := bestRank[a]; !ok {
		return BestNone
	}
	return a
}

// ItemResult is what happened on one item, the input of ItemOutcome.
type ItemResult struct {
	// FirstVerdict / FinalVerdict: verdict of the first and the last graded answer (same when answered once).
	FirstVerdict string
	FinalVerdict string
	// Aided: the final answer came after a narrowing question or an explanation.
	Aided bool
	// Revealed: the learner asked for the answer ('그냥 알려줘').
	Revealed bool
}

// ItemOutcome decides the single outcome of a finished item. An answer within
// MinCreditGap after a failure on the same objective (lastFailAt) counts as aided.
func ItemOutcome(r ItemResult, lastFailAt *time.Time, now time.Time) string {
	if r.Revealed {
		return OutcomeFailed
	}
	if r.FirstVerdict == VerdictHeld || r.FinalVerdict == VerdictHeld {
		return OutcomeHeld
	}
	recentFail := lastFailAt != nil && now.Sub(*lastFailAt) < MinCreditGap
	if r.Aided {
		if r.FinalVerdict == VerdictCorrect {
			return OutcomeAided
		}
		return OutcomeFailed
	}
	switch r.FinalVerdict {
	case VerdictCorrect:
		if recentFail {
			return OutcomeAided
		}
		return OutcomeUnaidedCorrect
	case VerdictPartial:
		if recentFail {
			return OutcomeAided
		}
		return OutcomeUnaidedPartial
	}
	return OutcomeFailed
}

// ApplyEvidence applies one item outcome to the objective (design §2.3 table, §2.6 graduation).
func ApplyEvidence(m Mastery, outcome string, now time.Time, intervals []int) Mastery {
	if m.Best == "" {
		m.Best = BestNone
	}
	switch outcome {
	case OutcomeUnaidedCorrect:
		m = applyUnaidedCorrect(m, now, intervals)
	case OutcomeUnaidedPartial:
		next := Due(max(m.Box-1, 0), now, intervals)
		m.NextReviewAt = &next
	case OutcomeAided, OutcomeFailed:
		best := BestAided
		if outcome == OutcomeFailed {
			best = BestRevealed
		}
		if m.Box >= 2 {
			m.Lapses++
		}
		next := Due(0, now, intervals)
		m.Box, m.GradStep, m.Best = 0, GradLadder, MaxBest(m.Best, best)
		m.LastFailAt, m.NextReviewAt = timePtr(now), &next
	default: // held (and anything unknown): no change
		return m
	}
	m.Stale = false
	return m
}

func applyUnaidedCorrect(m Mastery, now time.Time, intervals []int) Mastery {
	m.Best = BestUnaided
	if m.LastCreditAt != nil && now.Sub(*m.LastCreditAt) < MinCreditGap {
		m.Box = max(m.Box, 1)
		if m.NextReviewAt == nil && m.GradStep != GradDone {
			next := Due(m.Box, now, intervals)
			m.NextReviewAt = &next
		}
		return m
	}
	switch {
	case m.GradStep == GradPending:
		m.GradStep, m.NextReviewAt = GradDone, nil
	case m.GradStep == GradDone:
		m.NextReviewAt = nil
	case m.Box >= MaxBox:
		next := dueIn(GraduationDays, now)
		m.GradStep, m.NextReviewAt = GradPending, &next
	default:
		m.Box++
		next := Due(m.Box, now, intervals)
		m.NextReviewAt = &next
	}
	if m.Box >= 2 {
		m.Lapses = 0
	}
	m.LastCreditAt = timePtr(now)
	return m
}

// Restore undoes an item's evidence after an accepted dispute: back to the
// snapshot taken before it was applied, then best rises to unaided (the box
// does not). When other evidence reached the objective since (conflict), the
// current state is kept and only best is raised.
func Restore(current, prev Mastery, conflict bool) Mastery {
	out := prev
	if conflict {
		out = current
	}
	out.Best = MaxBest(out.Best, BestUnaided)
	return out
}

// KeepSince applies to a restored mastery (an accepted dispute without later
// evidence) what changed since the item without being evidence: a release from
// the inflow queue stays released (it does not wait again; a missing review
// date is the restored state's NextDue), an integrative credit and a stale mark
// stay. current is the state at the dispute.
func KeepSince(restored, current Mastery, now time.Time, intervals []int) Mastery {
	if restored.Queued && !current.Queued {
		restored.Queued = false
		if restored.NextReviewAt == nil && restored.GradStep != GradDone {
			due := NextDue(restored, now, intervals)
			restored.NextReviewAt = &due
		}
	}
	restored.IntegrativeOK = restored.IntegrativeOK || current.IntegrativeOK
	if current.Stale && !restored.Stale {
		restored = MarkStale(restored)
	}
	return restored
}

// MarkStale mirrors the SQL of a changed objective (design §2.5): the old
// evidence no longer proves the new text.
func MarkStale(m Mastery) Mastery {
	m.Stale, m.IntegrativeOK = true, false
	m.Box = min(m.Box, 1)
	m.GradStep = GradLadder
	return m
}

// Due is the next review time for a box: interval = intervals[clamp(box-1)]
// (box 0·1 → 1st, 2 → 2nd, 3 → 3rd), next = max(now+12h, KST day start + interval days).
func Due(box int, now time.Time, intervals []int) time.Time {
	if len(intervals) == 0 {
		intervals = DefaultReviewIntervals
	}
	i := min(max(box-1, 0), len(intervals)-1)
	return dueIn(intervals[i], now)
}

func dueIn(days int, now time.Time) time.Time {
	byDay := KSTDayStart(now).AddDate(0, 0, days)
	if soon := now.Add(MinCreditGap); soon.After(byDay) {
		return soon
	}
	return byDay
}

// DisplayLevel is the 0~5 level shown for an objective (design §2.2).
func DisplayLevel(m Mastery) int {
	switch {
	case m.Box >= MaxBox && m.IntegrativeOK:
		return LevelExplain
	case m.Box >= 2:
		return LevelRemember
	case m.Best == BestUnaided:
		return LevelAlone
	case m.Best == BestAided:
		return LevelFollowed
	case m.Best == BestRevealed:
		return LevelHeard
	}
	return LevelUnchecked
}

// ConceptLevel is the minimum level of the concept's active objectives (0 when none).
func ConceptLevel(levels []int) int {
	if len(levels) == 0 {
		return LevelUnchecked
	}
	low := levels[0]
	for _, l := range levels[1:] {
		low = min(low, l)
	}
	return low
}

// ConceptMastered: every active objective is at or above the target level.
func ConceptMastered(levels []int, target int) bool {
	return len(levels) > 0 && ConceptLevel(levels) >= target
}

func timePtr(t time.Time) *time.Time { return &t }
