package learning

import "fmt"

// Recommendation kinds, in priority order (design §3.5).
const (
	RecResume       = "resume"
	RecGoalPath     = "goal_path"
	RecStudyRequest = "study_request"
	RecEssentialGap = "essential_gap"
	RecNewGoal      = "new_goal"
)

// DefaultLectureMinutes is the nominal lecture length when no lecture exists
// yet (quality design v2.2 Q3: ~20 minutes; a stored lecture shows EstimateMinutes).
const DefaultLectureMinutes = 20

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
	// Review: the review queue and settings (TodayResult.Review; never affects Exhausted).
	Review ReviewInput
	// Lapsed: concepts with an objective lapsed RecheckLapses+ times without recovering (TodayResult.Recheck).
	Lapsed map[string]LapsedConcept
	// Unchecked: finished lectures never checked, most recent first (TodayResult.Unchecked).
	Unchecked []UncheckedLecture
	// LectureMinutes: EstimateMinutes of the lecture a recommendation would open, by concept
	// (absent = no lecture yet → DefaultLectureMinutes).
	LectureMinutes map[string]int
}

// LapsedConcept is how a "다시 확인" card opens: today's review when one of the
// lapsed objectives is due, else a check on the lecture the user last read ("" = none).
type LapsedConcept struct {
	Due       bool
	LectureID string
}

// RecheckCard is a "○○ 다시 확인" card shown before the goal path (design §5 P4).
type RecheckCard struct {
	ConceptID   string `json:"conceptId"`
	ConceptName string `json:"conceptName"`
	Area        string `json:"area"`
	GoalID      string `json:"goalId"`
	Due         bool   `json:"due"`
	LectureID   string `json:"lectureId,omitempty"`
}

// ReviewInput is the review summary input of BuildToday.
type ReviewInput struct {
	Counts        ReviewCounts
	ReviewMinutes int
	Grader        string
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
	// Review: due/backlog numbers and the time split "복습 n분 + 강의 30분" (separate from Items).
	Review ReviewToday `json:"review"`
	// Recheck: lapsed prerequisites of the goal paths' next concepts (separate from Items, never affects Exhausted).
	Recheck []RecheckCard `json:"recheck"`
	// Unchecked: "○○ 확인하기" for finished lectures with no review schedule yet (separate from Items, never affects Exhausted).
	Unchecked []UncheckedCard `json:"unchecked"`
	// Session: today's study session time "강의 n분 + 확인 n분 + 복습 n분" (SessionFor).
	Session SessionMinutes `json:"session"`
}

type todayBuilder struct {
	in    TodayInput
	used  map[string]bool
	items []Recommendation
	// recommended counts non-resume items: Limit caps what we suggest, never what the user was reading.
	recommended int
	an          Analysis
}

// learnable: active, not done, still has a pending objective.
func (b *todayBuilder) learnable(id string) (TodayConcept, bool) {
	c, ok := b.in.Concepts[id]
	if !ok || IsDone(NormalizeState(c.State)) || !c.HasPending || b.used[id] {
		return TodayConcept{}, false
	}
	return c, true
}

func (b *todayBuilder) full() bool { return b.recommended >= b.in.Limit }

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
		r.EstimatedMinutes = b.in.LectureMinutes[c.ID]
	}
	if r.EstimatedMinutes <= 0 {
		r.EstimatedMinutes = DefaultLectureMinutes
	}
	b.used[c.ID] = true
	if r.Kind != RecResume {
		b.recommended++
	}
	b.items = append(b.items, r)
}

// BuildToday orders recommendations: every in-progress lecture (not counted in Limit) → selected goal path's
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

	// 읽다 만 강의는 전부 보여 준다. 개념을 끝냈거나 '이미 앎'으로 표시했으면 더 읽을 이유가 없으니 빼고,
	// 지도에서 사라진 개념의 강의도 뺀다.
	// 읽고 있는 개념은 같은 개념을 또 추천하지 않도록 add 가 used 로 표시한다(의도).
	for _, r := range in.Resume {
		if c, ok := in.Concepts[r.ConceptID]; ok && !IsDone(NormalizeState(c.State)) {
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
	res := TodayResult{Items: b.items, Exhausted: b.recommended == 0,
		Review:  SummarizeReviews(in.Review.Counts, in.Review.ReviewMinutes, in.Review.Grader),
		Recheck: recheckCards(in, states), Unchecked: uncheckedCards(in)}
	res.Session = SessionFor(res.Items, res.Review)
	if res.Session.LectureMinutes > 0 {
		res.Review.LectureMinutes = res.Session.LectureMinutes
	}
	return res
}

func remainingMinutes(r ResumeItem) int {
	total := r.EstimatedMinutes
	if total <= 0 {
		total = DefaultLectureMinutes
	}
	left := total * (100 - r.Percent) / 100
	if left < 5 {
		left = min(5, total)
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
