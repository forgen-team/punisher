package learning

import "testing"

func TestWeakConcept(t *testing.T) {
	lv := func(level int, best string, lapses int) ObjectiveLevel {
		return ObjectiveLevel{Level: level, Best: best, Lapses: lapses}
	}
	cases := []struct {
		name string
		objs []ObjectiveLevel
		want bool
	}{
		{"no objectives", nil, false},
		{"completed but never checked (level 0, none)", []ObjectiveLevel{lv(0, BestNone, 0), lv(0, BestNone, 0)}, false},
		{"heard only", []ObjectiveLevel{lv(1, BestRevealed, 0)}, true},
		{"followed", []ObjectiveLevel{lv(2, BestAided, 0)}, true},
		{"alone on every objective", []ObjectiveLevel{lv(3, BestUnaided, 0), lv(4, BestUnaided, 0)}, false},
		{"one objective unchecked pulls the level to 0", []ObjectiveLevel{lv(4, BestUnaided, 0), lv(0, BestNone, 0)}, true},
		{"remembered but a lapse not recovered", []ObjectiveLevel{lv(4, BestUnaided, 0), lv(3, BestUnaided, 1)}, true},
		{"remembered everywhere", []ObjectiveLevel{lv(4, BestUnaided, 0), lv(5, BestUnaided, 0)}, false},
	}
	for _, c := range cases {
		if got := WeakConcept(c.objs); got != c.want {
			t.Errorf("%s: WeakConcept = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestValidMisconceptionStatus(t *testing.T) {
	for s, want := range map[string]bool{"open": true, "resolved": true, "": false, "closed": false} {
		if ValidMisconceptionStatus(s) != want {
			t.Errorf("%q", s)
		}
	}
}

// The "다시 확인" card: lapsed direct prerequisites of a goal path's next
// concept, once each, separate from Items (Exhausted unchanged).
func TestBuildTodayRecheck(t *testing.T) {
	concepts := map[string]TodayConcept{
		"a": {ID: "a", Name: "A", Area: "os", Depth: 1, State: "completed"},
		"b": {ID: "b", Name: "B", Area: "os", Depth: 1, State: "completed"},
		"c": {ID: "c", Name: "C", Area: "os", Depth: 2, State: "not_started", HasPending: true, PendingDepths: 1},
	}
	edges := []PrereqEdge{{From: "a", To: "c"}, {From: "b", To: "c"}}
	goals := []GoalInput{{ID: "g1", Title: "C까지", Path: []string{"a", "b", "c"}}, {ID: "g2", Title: "또 C", Path: []string{"a", "b", "c"}}}
	in := TodayInput{Concepts: concepts, Edges: edges, Goals: goals, Limit: 5}

	base := BuildToday(in)
	if len(base.Recheck) != 0 || base.Recheck == nil {
		t.Fatalf("no lapsed concept → empty (non-nil) recheck: %+v", base.Recheck)
	}
	in.Lapsed = map[string]LapsedConcept{"b": {Due: true}, "z": {LectureID: "l1"}}
	got := BuildToday(in)
	if len(got.Recheck) != 1 || got.Recheck[0].ConceptID != "b" || got.Recheck[0].GoalID != "g1" || !got.Recheck[0].Due || got.Recheck[0].ConceptName != "B" {
		t.Fatalf("recheck = %+v", got.Recheck)
	}
	if got.Exhausted != base.Exhausted || len(got.Items) != len(base.Items) || got.Items[0].Kind != RecGoalPath {
		t.Fatalf("items and Exhausted must not change: %+v vs %+v", got, base)
	}

	in.Concepts["c"] = TodayConcept{ID: "c", Name: "C", Area: "os", Depth: 2, State: "completed"}
	if done := BuildToday(in); len(done.Recheck) != 0 {
		t.Fatalf("a finished path has no next concept: %+v", done.Recheck)
	}
}
