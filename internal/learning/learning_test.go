package learning

import (
	"errors"
	"testing"
)

func TestNextState(t *testing.T) {
	cases := []struct {
		current, requested string
		want               string
		changed            bool
	}{
		{"", InProgress, InProgress, true},
		{NotStarted, Completed, Completed, true},
		{InProgress, Completed, Completed, true},
		{NotStarted, Known, Known, true},
		{Completed, Known, Known, true},
		// in_progress never downgrades a learned concept.
		{Completed, InProgress, Completed, false},
		{Known, InProgress, Known, false},
		// same state is a no-op.
		{Completed, Completed, Completed, false},
		// explicit reset is allowed.
		{Completed, NotStarted, NotStarted, true},
	}
	for _, tc := range cases {
		got, changed, err := NextState(tc.current, tc.requested)
		if err != nil {
			t.Fatalf("%s->%s: unexpected error %v", tc.current, tc.requested, err)
		}
		if got != tc.want || changed != tc.changed {
			t.Errorf("%q->%q = (%q,%v), want (%q,%v)", tc.current, tc.requested, got, changed, tc.want, tc.changed)
		}
	}
	if _, _, err := NextState(NotStarted, "mastered"); !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", err)
	}
}

// Graph used below (prerequisite edges, from → to):
//
//	process → container → kubernetes → pod
//	process → thread
//	memory  → oom ← process
func testGraph() ([]Node, []PrereqEdge) {
	nodes := []Node{
		{"os.process", 1}, {"os.thread", 1}, {"os.memory", 1}, {"os.oom", 2},
		{"infra.container", 1}, {"infra.kubernetes", 1}, {"infra.pod", 1},
	}
	edges := []PrereqEdge{
		{"os.process", "infra.container"},
		{"infra.container", "infra.kubernetes"},
		{"infra.kubernetes", "infra.pod"},
		{"os.process", "os.thread"},
		{"os.memory", "os.oom"},
		{"os.process", "os.oom"},
		{"retired.concept", "infra.pod"}, // endpoint not present: ignored
	}
	return nodes, edges
}

func frontierIDs(a Analysis) []string {
	ids := make([]string, 0, len(a.Frontier))
	for _, f := range a.Frontier {
		ids = append(ids, f.ID)
	}
	return ids
}

func TestAnalyzeNewUserFrontierIsRoots(t *testing.T) {
	nodes, edges := testGraph()
	a := Analyze(nodes, edges, nil)

	got := frontierIDs(a)
	// process unlocks container+thread (oom still needs memory) = 2; memory unlocks 0.
	want := []string{"os.process", "os.memory"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("frontier = %v, want %v", got, want)
	}
	if a.Nodes["os.process"].Unlocks != 2 {
		t.Fatalf("process unlocks = %d, want 2", a.Nodes["os.process"].Unlocks)
	}
	if !a.Nodes["infra.pod"].Locked {
		t.Fatalf("pod should be locked for a new user")
	}
}

func TestAnalyzeAfterCompletion(t *testing.T) {
	nodes, edges := testGraph()
	states := map[string]string{
		"os.process":      Completed,
		"infra.container": Known,
		"os.thread":       InProgress,
	}
	a := Analyze(nodes, edges, states)

	got := map[string]bool{}
	for _, id := range frontierIDs(a) {
		got[id] = true
	}
	for _, id := range []string{"infra.kubernetes", "os.thread", "os.memory"} {
		if !got[id] {
			t.Errorf("expected %s in frontier, got %v", id, frontierIDs(a))
		}
	}
	for _, id := range []string{"os.process", "infra.container", "infra.pod", "os.oom"} {
		if got[id] {
			t.Errorf("%s must not be in frontier", id)
		}
	}
	// memory is now oom's only missing prerequisite.
	if a.Nodes["os.memory"].Unlocks != 1 {
		t.Errorf("memory unlocks = %d, want 1", a.Nodes["os.memory"].Unlocks)
	}
	if a.Nodes["os.process"].Locked || a.Nodes["os.process"].Frontier {
		t.Errorf("done concept must be neither locked nor frontier")
	}
}

func TestObjectiveKeyIsCanonical(t *testing.T) {
	a := ObjectiveKey([]string{"infra.pod#2", "infra.pod#1", "infra.pod#2", " "})
	b := ObjectiveKey([]string{"infra.pod#1", "infra.pod#2"})
	if a != b || a != "infra.pod#1,infra.pod#2" {
		t.Fatalf("objective keys differ: %q vs %q", a, b)
	}
}

func TestDeriveIdempotencyKeyStable(t *testing.T) {
	k1 := DeriveIdempotencyKey(SourceRSS, "insight-1", "pod")
	k2 := DeriveIdempotencyKey(SourceRSS, " insight-1 ", "pod")
	k3 := DeriveIdempotencyKey(SourceManual, "insight-1", "pod")
	if k1 != k2 {
		t.Fatalf("whitespace in sourceRef should not change the key")
	}
	if k1 == k3 {
		t.Fatalf("different sources must produce different keys")
	}
}
