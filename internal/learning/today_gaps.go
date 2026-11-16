package learning

import (
	"fmt"
	"sort"

	"github.com/forgen-team/punisher/internal/curriculum"
)

// addEssentialGaps adds depth-1 frontier concepts, round-robin across areas
// starting from the area with the fewest learned concepts.
func (b *todayBuilder) addEssentialGaps() {
	doneByArea := map[string]int{}
	byArea := map[string][]TodayConcept{}
	for _, c := range b.in.Concepts {
		if IsDone(NormalizeState(c.State)) {
			doneByArea[c.Area]++
		}
	}
	for _, f := range b.an.Frontier { // already sorted by unlocks desc
		c, ok := b.learnable(f.ID)
		if ok && c.Depth == 1 && !c.Proposed {
			byArea[c.Area] = append(byArea[c.Area], c)
		}
	}
	areas := make([]string, 0, len(byArea))
	for a := range byArea {
		areas = append(areas, a)
	}
	sort.Slice(areas, func(i, j int) bool {
		if doneByArea[areas[i]] != doneByArea[areas[j]] {
			return doneByArea[areas[i]] < doneByArea[areas[j]]
		}
		return areas[i] < areas[j]
	})
	for round := 0; !b.full(); round++ {
		added := false
		for _, a := range areas {
			if b.full() {
				return
			}
			if round < len(byArea[a]) {
				c := byArea[a][round]
				b.add(c, Recommendation{Kind: RecEssentialGap, Reason: fmt.Sprintf(
					"상대적으로 덜 다룬 %s 영역의 공백을 채우는 필수 기본기입니다. 선행 개념을 모두 갖췄습니다.", curriculum.AreaTitle(c.Area))})
				added = true
			}
		}
		if !added {
			return
		}
	}
}

// addNewGoal suggests one deeper frontier concept as a new goal.
func (b *todayBuilder) addNewGoal() {
	if b.full() {
		return
	}
	for _, f := range b.an.Frontier {
		c, ok := b.learnable(f.ID)
		if !ok || c.Depth < 2 || c.Proposed {
			continue
		}
		b.add(c, Recommendation{Kind: RecNewGoal, Reason: fmt.Sprintf(
			"새 목표로 삼으면 %d개 개념이 이어서 열리는 심화 주제입니다. 기본기는 갖췄습니다.", f.Unlocks)})
		return
	}
}
