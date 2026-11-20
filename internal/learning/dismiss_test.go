package learning

import "testing"

// TestBuildTodaySkipsDismissed: a '필요 없음' concept is never recommended —
// not as a gap, not as 이어보기 — and leaves the next-chapter runway; its
// prerequisite role is unchanged (the dependent stays locked, so it is not
// recommended either).
func TestBuildTodaySkipsDismissed(t *testing.T) {
	in := TodayInput{
		Concepts: map[string]TodayConcept{
			"a.base": {ID: "a.base", Area: "os", Depth: 1, State: NotStarted, HasPending: true, PendingDepths: 1, Dismissed: true},
			"a.next": {ID: "a.next", Area: "os", Depth: 2, State: NotStarted, HasPending: true, PendingDepths: 1},
			"b.free": {ID: "b.free", Area: "database", Depth: 1, State: NotStarted, HasPending: true, PendingDepths: 1},
		},
		Edges:  []PrereqEdge{{From: "a.base", To: "a.next"}},
		Resume: []ResumeItem{{LectureID: "l1", ConceptID: "a.base", Percent: 40}},
		Limit:  5,
	}
	res := BuildToday(in)
	if len(res.Items) == 0 || res.Items[0].ConceptID != "b.free" {
		t.Fatalf("items = %+v, want b.free first (nothing dismissed ahead of it)", res.Items)
	}
	for _, it := range res.Items {
		if it.ConceptID == "a.base" || it.ConceptID == "a.next" {
			t.Errorf("recommended %s (%s): dismissed concept or its locked dependent", it.ConceptID, it.Kind)
		}
	}
	if got := RemainingMinutes(in.Concepts); got != 2*SessionMinutesPerLecture {
		t.Errorf("runway = %d, want the two non-dismissed concepts only", got)
	}
}
