package curriculum

import (
	"strings"
	"testing"

	seed "github.com/wooo-jin/punisher/curriculum"
)

func concept(id string, prereqs ...string) Concept {
	return Concept{
		ID:            id,
		Name:          id,
		Area:          "os",
		Depth:         1,
		Summary:       "요약",
		Objectives:    []Objective{{Text: "목표", Depth: 1}},
		Prerequisites: prereqs,
	}
}

func TestValidateDetectsCycle(t *testing.T) {
	concepts := []Concept{
		concept("os.a", "os.c"),
		concept("os.b", "os.a"),
		concept("os.c", "os.b"),
	}
	err := Validate(concepts)
	if err == nil || !strings.Contains(err.Error(), "prerequisite cycle") {
		t.Fatalf("expected cycle error, got %v", err)
	}
	cycle := FindPrerequisiteCycle(concepts)
	if len(cycle) != 4 || cycle[0] != cycle[len(cycle)-1] {
		t.Fatalf("expected closed 3-cycle path, got %v", cycle)
	}
}

func TestValidateAcceptsDAG(t *testing.T) {
	concepts := []Concept{
		concept("os.a"),
		concept("os.b", "os.a"),
		concept("os.c", "os.a", "os.b"),
	}
	if err := Validate(concepts); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateMissingPrerequisite(t *testing.T) {
	err := Validate([]Concept{concept("os.a", "os.missing")})
	if err == nil || !strings.Contains(err.Error(), `"os.missing" does not exist`) {
		t.Fatalf("expected missing prerequisite error, got %v", err)
	}
}

func TestValidateDuplicateAliasAfterNormalization(t *testing.T) {
	a := concept("infra.pod")
	a.Aliases = []string{"Pod", "파드"}
	b := concept("infra.other")
	b.Aliases = []string{"  pod "}
	err := Validate([]Concept{a, b})
	if err == nil || !strings.Contains(err.Error(), `alias "pod"`) {
		t.Fatalf("expected duplicate alias error, got %v", err)
	}
}

func TestValidateDuplicateID(t *testing.T) {
	err := Validate([]Concept{concept("os.a"), concept("os.a")})
	if err == nil || !strings.Contains(err.Error(), "duplicate concept id") {
		t.Fatalf("expected duplicate id error, got %v", err)
	}
}

func TestNormalizeAliasNFC(t *testing.T) {
	decomposed := "파드" // 파드 (NFD jamo)
	if NormalizeAlias(decomposed) != NormalizeAlias("파드") {
		t.Fatalf("NFD and NFC forms should normalize equally")
	}
	if NormalizeAlias("  Kubernetes   Pod ") != "kubernetes pod" {
		t.Fatalf("unexpected normalization: %q", NormalizeAlias("  Kubernetes   Pod "))
	}
}

func TestBuildVersionIsStable(t *testing.T) {
	first, err := Build([]Concept{concept("os.b", "os.a"), concept("os.a")})
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build([]Concept{concept("os.a"), concept("os.b", "os.a")})
	if err != nil {
		t.Fatal(err)
	}
	if first.Version != second.Version {
		t.Fatalf("version should not depend on input order")
	}
}

// TestRealSeedIsValid loads the embedded curriculum and checks the
// invariants the server relies on at startup.
func TestRealSeedIsValid(t *testing.T) {
	s, err := Load(seed.Files)
	if err != nil {
		t.Fatalf("real curriculum seed is invalid: %v", err)
	}
	if len(s.Concepts) < 140 {
		t.Fatalf("expected ~150 concepts, got %d", len(s.Concepts))
	}

	perArea := map[string]int{}
	for _, c := range s.Concepts {
		perArea[c.Area]++
	}
	for _, area := range Areas {
		if perArea[area] == 0 {
			t.Errorf("area %q has no concepts", area)
		}
	}

	ids := map[string]bool{}
	for _, c := range s.Concepts {
		ids[c.ID] = true
	}
	podPath := []string{
		"os.process", "os.exit-code-and-signals", "os.memory-oom",
		"infra.container", "infra.kubernetes", "infra.pod",
		"infra.pod-lifecycle", "infra.probes",
	}
	for _, id := range podPath {
		if !ids[id] {
			t.Errorf("required concept %q missing from seed", id)
		}
	}
	t.Logf("seed version=%s concepts=%d edges=%d aliases=%d perArea=%v",
		s.Version[:12], len(s.Concepts), len(s.Edges), len(s.Aliases), perArea)
}
