package learning

import (
	"errors"
	"testing"
	"time"
)

var processObjectives = []ObjectiveRef{
	{ID: "os.process#1", Depth: 1, Ordinal: 1},
	{ID: "os.process#2", Depth: 1, Ordinal: 2},
	{ID: "os.process#3", Depth: 2, Ordinal: 3},
}

func TestPlanLectureExcludesDoneObjectives(t *testing.T) {
	done := DoneSet{DoneKey("os.process#1", 1): true}
	plan, err := PlanLecture("os.process", processObjectives, done, 0)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Depth != 1 || plan.ObjectiveKey != "os.process#2" {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestCompletedOrKnownNeverRegenerated(t *testing.T) {
	done := DoneSet{}
	for _, o := range processObjectives {
		done[DoneKey(o.ID, o.Depth)] = true
	}
	for _, depth := range []int{0, 1, 2, 3} {
		if _, err := PlanLecture("os.process", processObjectives, done, depth); !errors.Is(err, ErrNothingToLearn) {
			t.Fatalf("depth %d: want ErrNothingToLearn, got %v", depth, err)
		}
	}
	if HasPending(processObjectives, done) {
		t.Fatal("no pending objectives expected")
	}
}

func TestDeeperLectureOnlyIfUnlearnedDepthExists(t *testing.T) {
	done := DoneSet{DoneKey("os.process#1", 1): true, DoneKey("os.process#2", 1): true}
	plan, err := PlanLecture("os.process", processObjectives, done, 2)
	if err != nil || plan.ObjectiveKey != "os.process#3" || plan.Depth != 2 {
		t.Fatalf("plan=%+v err=%v", plan, err)
	}
	// depth 3 has no objectives at all → refused.
	if _, err := PlanLecture("os.process", processObjectives, done, 3); !errors.Is(err, ErrNothingToLearn) {
		t.Fatalf("depth 3 must be refused, got %v", err)
	}
	// Learning depth-2 objective at depth 1 does not count for depth 2.
	partial := DoneSet{DoneKey("os.process#3", 1): true}
	if plan, err := PlanLecture("os.process", processObjectives, partial, 2); err != nil || plan.ObjectiveKey != "os.process#3" {
		t.Fatalf("depth-specific dedup: plan=%+v err=%v", plan, err)
	}
}

func TestAliasResolvesToSameObjectivesAndJobKey(t *testing.T) {
	index := AliasIndex{"프로세스": "os.process", "process": "os.process", "pid": "os.process", "os.process": "os.process"}
	day := KSTDay(time.Date(2026, 11, 6, 20, 0, 0, 0, time.UTC)) // 2026-11-07 KST
	if day != "2026-11-07" {
		t.Fatalf("KSTDay=%s", day)
	}
	var keys []string
	for _, q := range []string{"Process", "  프로세스 ", "PID", "os.process"} {
		id, ok := ResolveExact(index, q)
		if !ok || id != "os.process" {
			t.Fatalf("%q resolved to %q", q, id)
		}
		plan, err := PlanLecture(id, processObjectives, DoneSet{}, 0)
		if err != nil {
			t.Fatal(err)
		}
		keys = append(keys, LibraryJobKey(plan, day))
	}
	for _, k := range keys[1:] {
		if k != keys[0] {
			t.Fatalf("aliases must map to one idempotency key: %v", keys)
		}
	}
	// Objective-key order independence: a re-titled lecture with the same objectives is the same target.
	if ObjectiveKey([]string{"os.process#2", "os.process#1", "os.process#1"}) != "os.process#1,os.process#2" {
		t.Fatal("ObjectiveKey must be canonical")
	}
}

func TestBuildTodayNeverRecommendsDone(t *testing.T) {
	concepts := map[string]TodayConcept{
		"a": {ID: "a", Name: "A", Area: "os", Depth: 1, State: Completed, HasPending: false},
		"b": {ID: "b", Name: "B", Area: "os", Depth: 1, State: Known, HasPending: false},
		"c": {ID: "c", Name: "C", Area: "os", Depth: 1, HasPending: false}, // all objectives done via lectures
	}
	res := BuildToday(TodayInput{
		Concepts: concepts,
		Resume:   []ResumeItem{{LectureID: "l1", ConceptID: "a", Percent: 40}},
		Requests: []RequestInput{{ID: "r1", Query: "B", ConceptIDs: []string{"b"}}},
		Goals:    []GoalInput{{ID: "g", Title: "goal", Path: []string{"a", "b", "c"}}},
	})
	if !res.Exhausted || len(res.Items) != 0 {
		t.Fatalf("done concepts must not be recommended: %+v", res)
	}
}

func TestBuildTodayPriorityOrder(t *testing.T) {
	concepts := map[string]TodayConcept{
		"os.process":   {ID: "os.process", Name: "프로세스", Area: "os", Depth: 1, State: Completed},
		"os.container": {ID: "os.container", Name: "컨테이너", Area: "os", Depth: 2, HasPending: true},
		"infra.pod":    {ID: "infra.pod", Name: "Pod", Area: "infra-ops", Depth: 2, HasPending: true},
		"net.tcp":      {ID: "net.tcp", Name: "TCP", Area: "network-web", Depth: 1, HasPending: true},
		"db.index":     {ID: "db.index", Name: "인덱스", Area: "database", Depth: 1, HasPending: true},
		"ds.hash":      {ID: "ds.hash", Name: "해시", Area: "data-structures", Depth: 1, HasPending: true, State: InProgress},
		"net.http":     {ID: "net.http", Name: "HTTP", Area: "network-web", Depth: 2, HasPending: true},
		"dist.raft":    {ID: "dist.raft", Name: "Raft", Area: "distributed", Depth: 3, HasPending: true},
	}
	edges := []PrereqEdge{{"os.process", "os.container"}, {"os.container", "infra.pod"}, {"net.tcp", "net.http"}}
	depthOf := map[string]int{}
	for id, c := range concepts {
		depthOf[id] = c.Depth
	}
	res := BuildToday(TodayInput{
		Concepts: concepts, Edges: edges, Limit: 6,
		Resume:   []ResumeItem{{LectureID: "l-hash", ConceptID: "ds.hash", Percent: 50}},
		Goals:    []GoalInput{{ID: "g1", Title: "Pod 장애 이해", Path: PathTo("infra.pod", edges, depthOf)}},
		Requests: []RequestInput{{ID: "r1", Query: "HTTP keep-alive", ConceptIDs: []string{"net.http"}}},
	})
	want := []struct{ kind, id string }{
		{RecResume, "ds.hash"},
		{RecGoalPath, "os.container"},
		{RecStudyRequest, "net.tcp"}, // net.http locked → earliest prerequisite
		{RecEssentialGap, "db.index"},
		{RecNewGoal, "dist.raft"},
	}
	if len(res.Items) != len(want) {
		t.Fatalf("items=%+v", res.Items)
	}
	for i, w := range want {
		if res.Items[i].Kind != w.kind || res.Items[i].ConceptID != w.id {
			t.Fatalf("item %d = %s/%s, want %s/%s", i, res.Items[i].Kind, res.Items[i].ConceptID, w.kind, w.id)
		}
		if res.Items[i].Reason == "" || res.Items[i].EstimatedMinutes <= 0 {
			t.Fatalf("item %d lacks reason/minutes", i)
		}
	}
	if res.Exhausted {
		t.Fatal("not exhausted")
	}
}

func TestPathToAndStartingPoint(t *testing.T) {
	edges := []PrereqEdge{{"a", "c"}, {"b", "c"}, {"c", "d"}, {"x", "y"}}
	depthOf := map[string]int{"a": 1, "b": 1, "c": 2, "d": 3, "x": 1, "y": 1}
	path := PathTo("d", edges, depthOf)
	if len(path) != 4 || path[0] != "a" || path[1] != "b" || path[3] != "d" {
		t.Fatalf("path=%v", path)
	}
	if sp := StartingPoint(path, edges, map[string]string{"a": Completed}); sp != "b" {
		t.Fatalf("starting point=%q", sp)
	}
	if sp := StartingPoint(path, edges, map[string]string{"a": Known, "b": Completed, "c": Completed, "d": Completed}); sp != "" {
		t.Fatalf("all done should give empty, got %q", sp)
	}
}
