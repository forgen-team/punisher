package learning

import (
	"math"
	"time"
)

// FSRS memory model (W5a): each objective (and each review card) keeps a
// stability S (days until recall falls to 90%) and a difficulty D (1~10),
// updated after every judged answer. The next interval is the time recall is
// expected to fall to the target retention the user chose.
//
// Formulas and default weights are FSRS-4.5 (open-spaced-repetition, the
// defaults Anki ships for users without enough history to optimise) [확인:
// https://github.com/open-spaced-repetition/fsrs4anki/wiki/The-Algorithm].
// Personal optimisation is out of scope — logs here are small.
//
// Safe migration from the box ladder: the ladder still decides the level, the
// box and graduation. FSRS only re-times the next review, blended with the
// ladder interval by how many judgements fed it (BlendFSRS) — an objective
// with no FSRS history keeps its ladder interval exactly.

// Ratings fed to FSRS.
const (
	RatingAgain = 1
	RatingHard  = 2
	RatingGood  = 3
	RatingEasy  = 4
)

// Memory model choices (user_study_prefs.memory_model).
const (
	MemoryModelLadder = "ladder"
	MemoryModelFSRS   = "fsrs"
)

// Target retention bounds (user_study_prefs.target_retention).
const (
	DefaultTargetRetention = 0.9
	MinTargetRetention     = 0.8
	MaxTargetRetention     = 0.95
	// FSRSBlendReps: after this many FSRS judgements the FSRS interval is used alone.
	FSRSBlendReps = 4
	// MaxFSRSDays caps any interval (a year), like Anki's maximum interval default is far larger,
	// but objectives here graduate after the ladder anyway.
	MaxFSRSDays = 365
)

var fsrsW = [17]float64{0.4872, 1.4003, 3.7145, 13.8206, 5.1618, 1.2298, 0.8975, 0.031, 1.6474, 0.1367, 1.0461, 2.1072, 0.0793, 0.3246, 1.587, 0.2272, 2.8755}

const (
	fsrsDecay  = -0.5
	fsrsFactor = 19.0 / 81.0
)

// Memory is the FSRS state of one objective or card. Zero value = no history.
type Memory struct {
	Stability    float64    `json:"stability"`
	Difficulty   float64    `json:"difficulty"`
	Reps         int        `json:"reps"`
	LastReviewAt *time.Time `json:"lastReviewAt,omitempty"`
}

// ValidTargetRetention reports a retention inside the allowed range.
func ValidTargetRetention(r float64) bool {
	return r >= MinTargetRetention && r <= MaxTargetRetention
}

// ValidMemoryModel reports ladder | fsrs.
func ValidMemoryModel(m string) bool { return m == MemoryModelLadder || m == MemoryModelFSRS }

// RatingFor maps an item outcome to an FSRS rating (0 = not a judgement: held, recorded).
// Aided counts as Again like the ladder (box back to 0).
func RatingFor(outcome string) int {
	switch outcome {
	case OutcomeUnaidedCorrect:
		return RatingGood
	case OutcomeUnaidedPartial:
		return RatingHard
	case OutcomeAided, OutcomeFailed:
		return RatingAgain
	}
	return 0
}

// Retrievability is the predicted recall probability after elapsed days.
func Retrievability(stability, elapsedDays float64) float64 {
	if stability <= 0 {
		return 0
	}
	return math.Pow(1+fsrsFactor*elapsedDays/stability, fsrsDecay)
}

// FSRSInterval is the number of days until recall falls to retention (at least 1).
func FSRSInterval(stability, retention float64) int {
	if !ValidTargetRetention(retention) {
		retention = DefaultTargetRetention
	}
	days := stability / fsrsFactor * (math.Pow(retention, 1/fsrsDecay) - 1)
	return min(MaxFSRSDays, max(1, int(math.Round(days))))
}

func clampDifficulty(d float64) float64 { return math.Min(10, math.Max(1, d)) }

func initDifficulty(rating int) float64 {
	return clampDifficulty(fsrsW[4] - float64(rating-3)*fsrsW[5])
}

// NextMemory applies one rating at now. The first rating sets the initial state.
func NextMemory(m Memory, rating int, now time.Time) Memory {
	if rating < RatingAgain || rating > RatingEasy {
		return m
	}
	out := Memory{Reps: m.Reps + 1, LastReviewAt: &now}
	if m.Stability <= 0 || m.LastReviewAt == nil {
		out.Stability = fsrsW[rating-1]
		out.Difficulty = initDifficulty(rating)
		return out
	}
	elapsed := math.Max(0, now.Sub(*m.LastReviewAt).Hours()/24)
	r := Retrievability(m.Stability, elapsed)
	d := clampDifficulty(m.Difficulty)
	nextD := fsrsW[7]*initDifficulty(RatingGood) + (1-fsrsW[7])*(d-fsrsW[6]*float64(rating-3))
	out.Difficulty = clampDifficulty(nextD)
	if rating == RatingAgain {
		s := fsrsW[11] * math.Pow(d, -fsrsW[12]) * (math.Pow(m.Stability+1, fsrsW[13]) - 1) * math.Exp(fsrsW[14]*(1-r))
		out.Stability = math.Max(0.1, math.Min(s, m.Stability))
		return out
	}
	bonus := 1.0
	if rating == RatingHard {
		bonus = fsrsW[15]
	} else if rating == RatingEasy {
		bonus = fsrsW[16]
	}
	growth := math.Exp(fsrsW[8]) * (11 - d) * math.Pow(m.Stability, -fsrsW[9]) * (math.Exp(fsrsW[10]*(1-r)) - 1) * bonus
	out.Stability = m.Stability * (growth + 1)
	return out
}

// BlendDays mixes the ladder interval with the FSRS interval by how much FSRS
// history there is (reps of the state before this answer): 0 → ladder only,
// FSRSBlendReps or more → FSRS only. At least 1 day.
func BlendDays(ladderDays, fsrsDays, priorReps int) int {
	w := math.Min(1, float64(max(priorReps, 0))/FSRSBlendReps)
	return max(1, int(math.Round((1-w)*float64(ladderDays)+w*float64(fsrsDays))))
}

// RetimeReview re-times a ladder review date with FSRS (ladderDue is what the
// ladder chose). The result keeps the ladder's floor of 12 hours from now and
// starts at the KST day start like the ladder.
func RetimeReview(ladderDue, now time.Time, next Memory, priorReps int, retention float64) time.Time {
	ladderDays := int(math.Round(KSTDayStart(ladderDue).Sub(KSTDayStart(now)).Hours() / 24))
	days := BlendDays(max(ladderDays, 1), FSRSInterval(next.Stability, retention), priorReps)
	return dueIn(days, now)
}
