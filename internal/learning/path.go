package learning

import "sort"

// PathTo returns the prerequisite chain of target in learning order: every
// transitive prerequisite (topologically sorted, ties by depth then id) and
// finally target itself. Edges referencing ids outside depthOf are ignored.
func PathTo(target string, edges []PrereqEdge, depthOf map[string]int) []string {
	if _, ok := depthOf[target]; !ok {
		return nil
	}
	prereqs := map[string][]string{}
	for _, e := range edges {
		_, okFrom := depthOf[e.From]
		_, okTo := depthOf[e.To]
		if okFrom && okTo {
			prereqs[e.To] = append(prereqs[e.To], e.From)
		}
	}

	// Collect ancestors.
	inPath := map[string]bool{target: true}
	stack := []string{target}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		for _, p := range prereqs[id] {
			if !inPath[p] {
				inPath[p] = true
				stack = append(stack, p)
			}
		}
	}

	// Kahn's algorithm restricted to the path, deterministic tie-break.
	indeg := map[string]int{}
	dependents := map[string][]string{}
	for id := range inPath {
		for _, p := range prereqs[id] {
			if inPath[p] {
				indeg[id]++
				dependents[p] = append(dependents[p], id)
			}
		}
	}
	less := func(a, b string) bool {
		if depthOf[a] != depthOf[b] {
			return depthOf[a] < depthOf[b]
		}
		return a < b
	}
	ready := []string{}
	for id := range inPath {
		if indeg[id] == 0 {
			ready = append(ready, id)
		}
	}
	order := make([]string, 0, len(inPath))
	for len(ready) > 0 {
		sort.Slice(ready, func(i, j int) bool { return less(ready[i], ready[j]) })
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		for _, d := range dependents[id] {
			indeg[d]--
			if indeg[d] == 0 {
				ready = append(ready, d)
			}
		}
	}
	return order
}

// StartingPoint returns the earliest not-done concept on path whose
// prerequisites (within edges) are all done — "today's starting point".
// Returns "" when every concept on the path is done.
func StartingPoint(path []string, edges []PrereqEdge, states map[string]string) string {
	prereqs := map[string][]string{}
	for _, e := range edges {
		prereqs[e.To] = append(prereqs[e.To], e.From)
	}
	inPath := map[string]bool{}
	for _, id := range path {
		inPath[id] = true
	}
	for _, id := range path {
		if IsDone(NormalizeState(states[id])) {
			continue
		}
		ready := true
		for _, p := range prereqs[id] {
			if inPath[p] && !IsDone(NormalizeState(states[p])) {
				ready = false
				break
			}
		}
		if ready {
			return id
		}
	}
	return ""
}
