package learning

import (
	"math"
	"time"
)

// Adaptive intervals (v3 design §7): each objective carries an Ease that
// scales its review intervals. 2.0 leaves the ladder as the user set it.
const (
	EaseDefault = 2.0
	EaseMin     = 1.3
	EaseMax     = 2.5
)

// Learner confidence chosen before an answer (check_turns.learner_confidence, v3 §3).
const (
	ConfidenceSure    = "sure"
	ConfidenceUnsure  = "unsure"
	ConfidenceUnknown = "unknown"
)

// ValidConfidence: empty (not chosen) or one of the three values.
func ValidConfidence(c string) bool {
	return c == "" || c == ConfidenceSure || c == ConfidenceUnsure || c == ConfidenceUnknown
}

// ConfidentError: the learner was sure and the item failed ('착각'). It never
// changes the outcome; it marks the turn and lowers Ease more.
func ConfidentError(confidence, outcome string) bool {
	return confidence == ConfidenceSure && outcome == OutcomeFailed
}

// ScaledDays scales an interval by ease/2.0 (0 or out of range = neutral), at least 1 day.
func ScaledDays(days int, ease float64) int {
	if ease < EaseMin || ease > EaseMax {
		return days
	}
	return max(1, int(math.Round(float64(days)*ease/EaseDefault)))
}

// UpdateEase is the new ease after one judged item: only evidence that scored
// on the ladder moves it — a box rise (a fresh credit), an unaided partial and
// a failure. Repeats inside MinCreditGap, held, recorded and aided answers keep it.
func UpdateEase(ease float64, prev, next Mastery, outcome, confidence string) float64 {
	if ease < EaseMin || ease > EaseMax {
		ease = EaseDefault
	}
	switch outcome {
	case OutcomeUnaidedCorrect:
		if !credited(prev, next) {
			return ease
		}
		if confidence == ConfidenceSure {
			ease += 0.1
		} else {
			ease += 0.05
		}
	case OutcomeUnaidedPartial:
		ease -= 0.05
	case OutcomeFailed:
		if ConfidentError(confidence, outcome) {
			ease -= 0.3
		} else {
			ease -= 0.2
		}
	default:
		return ease
	}
	return math.Round(min(EaseMax, max(EaseMin, ease))*100) / 100
}

func credited(prev, next Mastery) bool {
	if next.LastCreditAt == nil {
		return false
	}
	return prev.LastCreditAt == nil || next.LastCreditAt.After(*prev.LastCreditAt)
}

// PullDue brings an objective's next review forward to Due(0) after a failed
// synthesis item (v3 §5): box, mark and lapses stay. Objectives without a
// review date (graduated, waiting in the inflow queue, in known-verification)
// are left alone.
func PullDue(m Mastery, now time.Time, intervals []int) Mastery {
	if m.NextReviewAt == nil || m.GradStep == GradDone || m.Queued || m.Verify {
		return m
	}
	soon := m.due(0, now, intervals)
	if soon.Before(*m.NextReviewAt) {
		m.NextReviewAt = &soon
	}
	return m
}
