package learning

import "sort"

// Node is an active concept in the graph.
type Node struct {
	ID    string
	Depth int
}

// PrereqEdge means From must be learned before To.
type PrereqEdge struct {
	From string
	To   string
}

// NodeAnalysis is the per-user view of one concept.
type NodeAnalysis struct {
	State string
	// Locked: not done and at least one prerequisite is not done.
	Locked bool
	// Frontier: not done and every prerequisite is done (roots included).
	Frontier bool
	// Unlocks: how many not-done dependents would become unlocked if this
	// concept were done (i.e. this is their only missing prerequisite).
	Unlocks int
}

// FrontierEntry is a recommended next concept.
type FrontierEntry struct {
	ID      string
	Depth   int
	State   string
	Unlocks int
}

// Analysis is the result of Analyze.
type Analysis struct {
	Nodes    map[string]NodeAnalysis
	Frontier []FrontierEntry // sorted: unlocks desc, depth asc, id asc
}

// Analyze computes lock/frontier/unlock information for a user. Edges whose
// endpoints are not in nodes (e.g. retired concepts) are ignored, so a
// retired prerequisite never blocks progress.
func Analyze(nodes []Node, edges []PrereqEdge, states map[string]string) Analysis {
	present := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		present[n.ID] = n
	}
	prereqs := make(map[string][]string)
	dependents := make(map[string][]string)
	for _, e := range edges {
		if _, ok := present[e.From]; !ok {
			continue
		}
		if _, ok := present[e.To]; !ok {
			continue
		}
		prereqs[e.To] = append(prereqs[e.To], e.From)
		dependents[e.From] = append(dependents[e.From], e.To)
	}

	stateOf := func(id string) string { return NormalizeState(states[id]) }
	missing := func(id string) int {
		count := 0
		for _, p := range prereqs[id] {
			if !IsDone(stateOf(p)) {
				count++
			}
		}
		return count
	}

	out := Analysis{Nodes: make(map[string]NodeAnalysis, len(nodes))}
	for _, n := range nodes {
		state := stateOf(n.ID)
		done := IsDone(state)
		a := NodeAnalysis{State: state}
		if !done {
			a.Locked = missing(n.ID) > 0
			a.Frontier = !a.Locked
			for _, d := range dependents[n.ID] {
				// n is not done, so it is one of d's missing prerequisites;
				// d unlocks when n is the only one.
				if !IsDone(stateOf(d)) && missing(d) == 1 {
					a.Unlocks++
				}
			}
		}
		out.Nodes[n.ID] = a
		if a.Frontier {
			out.Frontier = append(out.Frontier, FrontierEntry{ID: n.ID, Depth: n.Depth, State: state, Unlocks: a.Unlocks})
		}
	}

	sort.Slice(out.Frontier, func(i, j int) bool {
		a, b := out.Frontier[i], out.Frontier[j]
		if a.Unlocks != b.Unlocks {
			return a.Unlocks > b.Unlocks
		}
		if a.Depth != b.Depth {
			return a.Depth < b.Depth
		}
		return a.ID < b.ID
	})
	return out
}
