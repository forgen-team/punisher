package learning

import (
	"fmt"
	"sort"

	"github.com/wooo-jin/punisher/internal/curriculum"
)

// Recommendation kinds, in priority order (design §3.5).
const (
	RecResume       = "resume"
	RecGoalPath     = "goal_path"
	RecStudyRequest = "study_request"
	RecEssentialGap = "essential_gap"
	RecNewGoal      = "new_goal"
)

// DefaultLectureMinutes is the nominal lecture length.
const DefaultLectureMinutes = 30

// TodayConcept is the per-concept input for recommendations.
type TodayConcept struct {
	ID            string
	Name          string
	Area          string
	Depth         int
	Summary       string
	PracticalCase string
	State         string
	// HasPending: at least one active objective is not completed/known.
	HasPending bool
	// Proposed: dynamic concept not yet confirmed; only reachable via study requests.
	Proposed bool
	// Private: a user's chapter concept (never pre-generated as a shared library lecture).
	Private bool
	// PendingDepths: depths that still have a pending objective (≈ lectures left).
	PendingDepths int
}

// ResumeItem is an in-progress lecture.
type ResumeItem struct {
	LectureID        string
	ConceptID        string
	Title            string
	Anchor           string
	Percent          int
	EstimatedMinutes int
	// ParentLectureID: lecture it was opened from via "더 깊게" ("" = none).
	ParentLectureID string
}

// GoalInput is an active goal with its prerequisite path (PathTo order).
type GoalInput struct {
	ID    string
	Title string
	Path  []string
}

// RequestInput is a resolved, pending study request.
type RequestInput struct {
	ID         string
	Query      string
	ConceptIDs []string
}

// TodayInput is everything BuildToday needs; it performs no I/O.
type TodayInput struct {
	Concepts map[string]TodayConcept // active concepts
	Edges    []PrereqEdge
	Resume   []ResumeItem   // most recent first
	Goals    []GoalInput    // priority order
	Requests []RequestInput // oldest first
	Limit    int
}

// Recommendation is one "today" item.
type Recommendation struct {
	Kind               string `json:"kind"`
	ConceptID          string `json:"conceptId"`
	ConceptName        string `json:"conceptName"`
	Area               string `json:"area"`
	Depth              int    `json:"depth"`
	LectureID          string `json:"lectureId,omitempty"`
	GoalID             string `json:"goalId,omitempty"`
	StudyRequestID     string `json:"studyRequestId,omitempty"`
	Title              string `json:"title"`
	Reason             string `json:"reason"`
	PracticalSituation string `json:"practicalSituation"`
	EstimatedMinutes   int    `json:"estimatedMinutes"`
	Anchor             string `json:"anchor,omitempty"`
	Percent            int    `json:"percent,omitempty"`
	// ParentLectureID (resume only): the lecture this one was opened from via "더 깊게".
	ParentLectureID string `json:"parentLectureId,omitempty"`
}

// TodayResult is the GET /v1/today payload core.
type TodayResult struct {
	Items     []Recommendation `json:"items"`
	Exhausted bool             `json:"exhausted"`
	// NextChapter is set when the remaining learnable runway is short (see CheckNextChapter).
	NextChapter *NextChapterPrompt `json:"nextChapter,omitempty"`
}

type todayBuilder struct {
	in    TodayInput
	used  map[string]bool
	items []Recommendation
	an    Analysis
}

// learnable: active, not done, still has a pending objective.
func (b *todayBuilder) learnable(id string) (TodayConcept, bool) {
	c, ok := b.in.Concepts[id]
	if !ok || IsDone(NormalizeState(c.State)) || !c.HasPending || b.used[id] {
		return TodayConcept{}, false
	}
	return c, true
}

func (b *todayBuilder) full() bool { return len(b.items) >= b.in.Limit }

func (b *todayBuilder) add(c TodayConcept, r Recommendation) {
	r.ConceptID, r.ConceptName, r.Area, r.Depth = c.ID, c.Name, c.Area, c.Depth
	if r.Title == "" {
		r.Title = c.Name
	}
	if r.PracticalSituation == "" {
		r.PracticalSituation = c.PracticalCase
	}
	if r.PracticalSituation == "" {
		r.PracticalSituation = c.Summary
	}
	if r.EstimatedMinutes <= 0 {
		r.EstimatedMinutes = DefaultLectureMinutes
	}
	b.used[c.ID] = true
	b.items = append(b.items, r)
}

// BuildToday orders recommendations: resume in-progress → selected goal path's
// next unlearned prerequisite → pending study requests → essential gaps
// (depth-1 frontier, balanced across areas) → one new deeper-goal suggestion.
// Completed/known concepts and concepts without pending objectives are never
// recommended. If nothing qualifies the result is Exhausted.
func BuildToday(in TodayInput) TodayResult {
	if in.Limit <= 0 {
		in.Limit = 5
	}
	nodes := make([]Node, 0, len(in.Concepts))
	states := make(map[string]string, len(in.Concepts))
	for _, c := range in.Concepts {
		nodes = append(nodes, Node{ID: c.ID, Depth: c.Depth})
		states[c.ID] = c.State
	}
	b := &todayBuilder{in: in, used: map[string]bool{}, an: Analyze(nodes, in.Edges, states)}

	for _, r := range in.Resume {
		if b.full() {
			break
		}
		if c, ok := b.learnable(r.ConceptID); ok {
			b.add(c, Recommendation{Kind: RecResume, LectureID: r.LectureID, Title: r.Title, Anchor: r.Anchor,
				Percent: r.Percent, EstimatedMinutes: remainingMinutes(r), ParentLectureID: r.ParentLectureID,
				Reason: fmt.Sprintf("지난번 %d%%까지 본 강의입니다. 멈춘 위치에서 이어서 끝내면 흐름이 끊기지 않습니다.", r.Percent)})
		}
	}
	for _, g := range in.Goals {
		if b.full() {
			break
		}
		next := StartingPoint(g.Path, in.Edges, states)
		if c, ok := b.learnable(next); ok {
			b.add(c, Recommendation{Kind: RecGoalPath, GoalID: g.ID,
				Reason: fmt.Sprintf("목표 '%s'로 가는 경로에서 아직 배우지 않은 가장 앞선 선행 개념입니다.", g.Title)})
		}
	}
	for _, r := range in.Requests {
		if b.full() {
			break
		}
		b.addRequest(r)
	}
	b.addEssentialGaps()
	b.addNewGoal()

	if b.items == nil {
		b.items = []Recommendation{}
	}
	return TodayResult{Items: b.items, Exhausted: len(b.items) == 0}
}

func remainingMinutes(r ResumeItem) int {
	total := r.EstimatedMinutes
	if total <= 0 {
		total = DefaultLectureMinutes
	}
	left := total * (100 - r.Percent) / 100
	if left < 5 {
		left = 5
	}
	return left
}

// addRequest recommends the requested concept, or its earliest unlearned
// prerequisite when it is still locked.
func (b *todayBuilder) addRequest(r RequestInput) {
	depthOf := make(map[string]int, len(b.in.Concepts))
	for id, c := range b.in.Concepts {
		depthOf[id] = c.Depth
	}
	states := make(map[string]string, len(b.in.Concepts))
	for id, c := range b.in.Concepts {
		states[id] = c.State
	}
	for _, target := range r.ConceptIDs {
		next := StartingPoint(PathTo(target, b.in.Edges, depthOf), b.in.Edges, states)
		c, ok := b.learnable(next)
		if !ok {
			continue
		}
		reason := fmt.Sprintf("학습 요청 '%s'에 답하는 개념입니다.", r.Query)
		if next != target {
			reason = fmt.Sprintf("학습 요청 '%s'을(를) 이해하려면 먼저 필요한 선행 개념입니다.", r.Query)
		}
		b.add(c, Recommendation{Kind: RecStudyRequest, StudyRequestID: r.ID, Reason: reason})
		return
	}
}

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
					"선행 개념을 모두 갖춘 필수 기본기입니다. 상대적으로 덜 다룬 %s 영역의 공백을 채웁니다.", curriculum.AreaTitle(c.Area))})
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
			"기본기를 갖춘 심화 주제입니다. 새 목표로 삼으면 %d개 개념이 이어서 열립니다.", f.Unlocks)})
		return
	}
}
