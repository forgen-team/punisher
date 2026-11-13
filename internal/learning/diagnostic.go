package learning

import (
	"sort"
	"time"
)

// KnownVerifyPerDay: known verifications mixed into a day's reviews (design §5 P3: 1~2).
const KnownVerifyPerDay = 2

// Placement diagnostic bounds (design §5 P3: 3~5 items).
const (
	MinPlacementItems = 3
	MaxPlacementItems = 5
)

// ApplyKnownVerify applies the outcome of a '이미 앎' verification: a pass is
// best=unaided with box ≥ 1 (a fresh objective enters the ladder at box 1);
// any other judged outcome is ordinary evidence (ApplyEvidence). Either way the
// objective leaves the verification queue; held changes nothing.
func ApplyKnownVerify(m Mastery, outcome string, now time.Time, intervals []int) Mastery {
	if outcome == OutcomeHeld {
		return m
	}
	m.Verify = false
	if outcome != OutcomeUnaidedCorrect {
		return ApplyEvidence(m, outcome, now, intervals)
	}
	m.Best, m.Stale = BestUnaided, false
	if m.Box < 1 {
		m.Box, m.LastCreditAt = 1, timePtr(now)
		next := Due(1, now, intervals)
		m.NextReviewAt = &next
	} else if m.NextReviewAt == nil && m.GradStep != GradDone {
		next := Due(m.Box, now, intervals)
		m.NextReviewAt = &next
	}
	return m
}

// SuggestRelearn: a verification that did not pass (judged, not unaided
// correct) offers '다시 배우기' (design §2.4).
func SuggestRelearn(outcome string) bool {
	return outcome != OutcomeUnaidedCorrect && outcome != OutcomeHeld && outcome != OutcomeRecorded && outcome != ""
}

// MixKnownVerify splits a session's item limit between due reviews and known
// verifications: up to KnownVerifyPerDay verifications, but when the limit is
// that small a due review keeps at least one slot.
func MixKnownVerify(limit, due, verifyReady int) (reviews, verify int) {
	if limit <= 0 {
		return 0, 0
	}
	verify = min(KnownVerifyPerDay, max(verifyReady, 0), limit)
	if due > 0 && verify >= limit {
		verify = limit - 1
	}
	reviews = min(max(due, 0), limit-verify)
	return reviews, verify
}

// InterleaveVerify spreads verification items evenly through the review items
// (never two in a row when there are enough reviews).
func InterleaveVerify(reviews, verify []string) []string {
	out := make([]string, 0, len(reviews)+len(verify))
	if len(verify) == 0 {
		return append(out, reviews...)
	}
	step := float64(len(reviews)) / float64(len(verify)+1)
	r := 0
	for i, v := range verify {
		upto := int(step*float64(i+1) + 0.5)
		for ; r < upto && r < len(reviews); r++ {
			out = append(out, reviews[r])
		}
		out = append(out, v)
	}
	return append(out, reviews[r:]...)
}

// PlacementConcept is a chapter concept that can be diagnosed.
type PlacementConcept struct {
	ConceptID   string
	ObjectiveID string // the concept's first active objective
	Depth       int
	// Order: position in the chapter (prerequisites first).
	Order int
	State string
}

// PickPlacement orders the chapter's not-yet-learned concepts easy → hard
// (depth, then chapter order) and picks MaxPlacementItems spread evenly across
// that range. Fewer than MinPlacementItems candidates → none (no diagnostic).
func PickPlacement(list []PlacementConcept) []PlacementConcept {
	open := []PlacementConcept{}
	for _, c := range list {
		if !IsDone(c.State) && c.ObjectiveID != "" {
			open = append(open, c)
		}
	}
	if len(open) < MinPlacementItems {
		return nil
	}
	sort.SliceStable(open, func(i, j int) bool {
		if open[i].Depth != open[j].Depth {
			return open[i].Depth < open[j].Depth
		}
		return open[i].Order < open[j].Order
	})
	if len(open) <= MaxPlacementItems {
		return open
	}
	out := make([]PlacementConcept, 0, MaxPlacementItems)
	last := len(open) - 1
	for i := 0; i < MaxPlacementItems; i++ {
		out = append(out, open[(i*last+(MaxPlacementItems-1)/2)/(MaxPlacementItems-1)])
	}
	return out
}
