package learning

import (
	"reflect"
	"testing"
	"time"
)

func TestSynthesisEligible(t *testing.T) {
	now := time.Date(2026, 11, 16, 1, 0, 0, 0, time.UTC)
	three := []SynthesisCandidate{
		{ObjectiveID: "o1", ConceptID: "a", Area: "network"},
		{ObjectiveID: "o2", ConceptID: "b", Area: "network"},
		{ObjectiveID: "o3", ConceptID: "c", Area: "os"},
	}
	if !SynthesisEligible(three, nil, now) {
		t.Fatal("3 concepts in 2 areas should be eligible")
	}
	recent := now.Add(-6 * 24 * time.Hour)
	if SynthesisEligible(three, &recent, now) {
		t.Fatal("a synthesis within a week blocks the next")
	}
	week := now.Add(-7 * 24 * time.Hour)
	if !SynthesisEligible(three, &week, now) {
		t.Fatal("a week later it is eligible again")
	}
	oneArea := []SynthesisCandidate{three[0], three[1], {ObjectiveID: "o4", ConceptID: "d", Area: "network"}}
	if SynthesisEligible(oneArea, nil, now) {
		t.Fatal("one area is not enough")
	}
	if SynthesisEligible(three[:2], nil, now) {
		t.Fatal("two concepts are not enough")
	}
}

func TestPickSynthesis(t *testing.T) {
	day := func(d int) *time.Time {
		v := time.Date(2026, 11, d, 0, 0, 0, 0, time.UTC)
		return &v
	}
	cands := []SynthesisCandidate{
		{ObjectiveID: "tcp-1", ConceptID: "tcp", Area: "network", LastCreditAt: day(1)},
		{ObjectiveID: "tcp-2", ConceptID: "tcp", Area: "network", LastCreditAt: day(3)},
		{ObjectiveID: "tcp-3", ConceptID: "tcp", Area: "network", LastCreditAt: day(2)},
		{ObjectiveID: "dns-1", ConceptID: "dns", Area: "network", LastCreditAt: day(2)},
		{ObjectiveID: "proc-1", ConceptID: "process", Area: "os", LastCreditAt: day(5)},
		{ObjectiveID: "idx-1", ConceptID: "index", Area: "db", LastCreditAt: day(4)},
		{ObjectiveID: "tx-1", ConceptID: "tx", Area: "db", LastCreditAt: day(6), IntegrativeOK: true},
	}
	links := map[string][]string{"tcp": {"process"}, "index": {"tx"}}
	got := PickSynthesis(cands, links, "u:2026-W47")
	// tcp (oldest) → index (oldest of another area) → linked: process, tx.
	want := SynthesisPick{
		ConceptIDs:   []string{"tcp", "index", "process", "tx"},
		ObjectiveIDs: []string{"tcp-1", "tcp-3", "idx-1", "proc-1", "tx-1"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("pick = %+v\nwant %+v", got, want)
	}

	// No links: fill to the minimum with the next oldest.
	got = PickSynthesis(cands, nil, "u:2026-W47")
	if !reflect.DeepEqual(got.ConceptIDs, []string{"tcp", "index", "dns"}) {
		t.Fatalf("no links = %v", got.ConceptIDs)
	}
	if !reflect.DeepEqual(PickSynthesis(cands, nil, "u:2026-W47"), got) {
		t.Fatal("pick must be deterministic")
	}
	if p := PickSynthesis(cands[:4], nil, "s"); len(p.ConceptIDs) != 0 {
		t.Fatalf("two concepts must give an empty pick: %+v", p)
	}
}

func TestSynthesisWeekSeed(t *testing.T) {
	// 2026-11-15 23:30 UTC = 11-16 08:30 KST (Monday, ISO week 47).
	if s := SynthesisWeekSeed("u", time.Date(2026, 11, 15, 23, 30, 0, 0, time.UTC)); s != "u:2026-W47" {
		t.Fatalf("seed = %q", s)
	}
}
