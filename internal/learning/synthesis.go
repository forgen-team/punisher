package learning

import (
	"fmt"
	"hash/fnv"
	"sort"
	"time"
)

// Synthesis (v3 design §5): a weekly item set that mixes 3~4 checked concepts
// from at least two areas. Results never move boxes: a defended answer sets
// integrative_ok on the covered objectives, a failure pulls their reviews forward.
const (
	SynthesisMinConcepts = 3
	SynthesisMaxConcepts = 4
	SynthesisMinAreas    = 2
	SynthesisEvery       = 7 * 24 * time.Hour
	// SynthesisObjectivesPerConcept caps the objectives one concept brings in.
	SynthesisObjectivesPerConcept = 2
)

// SynthesisCandidate is one objective with mastery evidence from a check.
type SynthesisCandidate struct {
	ObjectiveID   string
	ConceptID     string
	Area          string
	LastCreditAt  *time.Time
	IntegrativeOK bool
}

// SynthesisPick is the chosen set: concepts in order (first = the session's
// representative concept) and the objectives the items cover.
type SynthesisPick struct {
	ConceptIDs   []string
	ObjectiveIDs []string
}

// SynthesisEligible: enough checked concepts across enough areas, and no
// synthesis session in the last week.
func SynthesisEligible(cands []SynthesisCandidate, lastSynthesisAt *time.Time, now time.Time) bool {
	if lastSynthesisAt != nil && now.Sub(*lastSynthesisAt) < SynthesisEvery {
		return false
	}
	concepts, areas := map[string]bool{}, map[string]bool{}
	for _, c := range cands {
		concepts[c.ConceptID], areas[c.Area] = true, true
	}
	return len(concepts) >= SynthesisMinConcepts && len(areas) >= SynthesisMinAreas
}

type synthConcept struct {
	id, area string
	oldest   *time.Time // oldest credit among objectives still without integrative_ok
	pending  bool       // has an objective without integrative_ok
	order    uint64     // deterministic tie-break
	objs     []SynthesisCandidate
}

// PickSynthesis picks the week's concepts: the two that were credited longest
// ago (objectives already integrative-ok count last; the second from another
// area when possible), then concepts linked to them (prerequisite/related),
// then any other area, up to SynthesisMaxConcepts. seed (user + ISO week)
// makes ties deterministic. Returns an empty pick when not enough concepts.
func PickSynthesis(cands []SynthesisCandidate, links map[string][]string, seed string) SynthesisPick {
	concepts := groupSynthesis(cands, seed)
	if len(concepts) < SynthesisMinConcepts {
		return SynthesisPick{}
	}
	var picked []*synthConcept
	used := map[string]bool{}
	take := func(c *synthConcept) {
		picked, used[c.id] = append(picked, c), true
	}
	take(concepts[0])
	if c := firstWhere(concepts, used, func(c *synthConcept) bool { return c.area != picked[0].area }); c != nil {
		take(c)
	} else if c := firstWhere(concepts, used, func(*synthConcept) bool { return true }); c != nil {
		take(c)
	}
	linked := map[string]bool{}
	for _, p := range picked {
		for _, id := range links[p.id] {
			linked[id] = true
		}
	}
	for len(picked) < SynthesisMaxConcepts {
		c := firstWhere(concepts, used, func(c *synthConcept) bool { return linked[c.id] })
		if c == nil {
			break
		}
		take(c)
	}
	for len(picked) < SynthesisMinConcepts {
		c := firstWhere(concepts, used, func(*synthConcept) bool { return true })
		if c == nil {
			break
		}
		take(c)
	}
	out := SynthesisPick{}
	for _, c := range picked {
		out.ConceptIDs = append(out.ConceptIDs, c.id)
		for i, o := range c.objs {
			if i == SynthesisObjectivesPerConcept {
				break
			}
			out.ObjectiveIDs = append(out.ObjectiveIDs, o.ObjectiveID)
		}
	}
	return out
}

func groupSynthesis(cands []SynthesisCandidate, seed string) []*synthConcept {
	byID := map[string]*synthConcept{}
	var list []*synthConcept
	for _, c := range cands {
		g := byID[c.ConceptID]
		if g == nil {
			g = &synthConcept{id: c.ConceptID, area: c.Area, order: seedOrder(seed, c.ConceptID)}
			byID[c.ConceptID] = g
			list = append(list, g)
		}
		g.objs = append(g.objs, c)
		if !c.IntegrativeOK {
			g.pending = true
			if c.LastCreditAt != nil && (g.oldest == nil || c.LastCreditAt.Before(*g.oldest)) {
				g.oldest = c.LastCreditAt
			}
		}
	}
	for _, g := range list {
		sort.SliceStable(g.objs, func(i, j int) bool { return objBefore(g.objs[i], g.objs[j]) })
	}
	sort.SliceStable(list, func(i, j int) bool {
		a, b := list[i], list[j]
		if a.pending != b.pending {
			return a.pending
		}
		if !timeEqual(a.oldest, b.oldest) {
			return timeBefore(a.oldest, b.oldest)
		}
		return a.order < b.order
	})
	return list
}

func objBefore(a, b SynthesisCandidate) bool {
	if a.IntegrativeOK != b.IntegrativeOK {
		return !a.IntegrativeOK
	}
	if !timeEqual(a.LastCreditAt, b.LastCreditAt) {
		return timeBefore(a.LastCreditAt, b.LastCreditAt)
	}
	return a.ObjectiveID < b.ObjectiveID
}

func firstWhere(list []*synthConcept, used map[string]bool, ok func(*synthConcept) bool) *synthConcept {
	for _, c := range list {
		if !used[c.id] && ok(c) {
			return c
		}
	}
	return nil
}

// timeBefore orders nil (never credited) first: it has waited longest.
func timeBefore(a, b *time.Time) bool {
	switch {
	case a == nil:
		return b != nil
	case b == nil:
		return false
	}
	return a.Before(*b)
}

func timeEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Equal(*b)
}

func seedOrder(seed, id string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(seed + "\x00" + id))
	return h.Sum64()
}

// SynthesisWeekSeed is the tie-break seed of a user's ISO week (KST).
func SynthesisWeekSeed(userID string, now time.Time) string {
	y, w := now.In(kst).ISOWeek()
	return fmt.Sprintf("%s:%d-W%02d", userID, y, w)
}
