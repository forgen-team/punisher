package learning

import "sort"

// recheckCards: for each goal path's next concept, its direct prerequisites
// that lapsed without recovering (in.Lapsed), once per concept, in goal order.
func recheckCards(in TodayInput, states map[string]string) []RecheckCard {
	out := []RecheckCard{}
	if len(in.Lapsed) == 0 {
		return out
	}
	prereqs := map[string][]string{}
	for _, e := range in.Edges {
		prereqs[e.To] = append(prereqs[e.To], e.From)
	}
	seen := map[string]bool{}
	for _, g := range in.Goals {
		next := StartingPoint(g.Path, in.Edges, states)
		if next == "" {
			continue
		}
		ps := append([]string(nil), prereqs[next]...)
		sort.Strings(ps)
		for _, p := range ps {
			lapsed, ok := in.Lapsed[p]
			c, known := in.Concepts[p]
			if !ok || !known || seen[p] {
				continue
			}
			seen[p] = true
			out = append(out, RecheckCard{ConceptID: p, ConceptName: c.Name, Area: c.Area, GoalID: g.ID,
				Due: lapsed.Due, LectureID: lapsed.LectureID})
		}
	}
	return out
}
