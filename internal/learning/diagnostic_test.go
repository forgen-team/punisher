package learning

import (
	"strings"
	"testing"
	"time"
)

func TestApplyKnownVerify(t *testing.T) {
	now := time.Date(2026, 11, 13, 10, 0, 0, 0, kst)
	fresh := Mastery{Best: BestNone, Verify: true}
	pass := ApplyKnownVerify(fresh, OutcomeUnaidedCorrect, now, nil)
	if pass.Box != 1 || pass.Best != BestUnaided || pass.Verify || pass.NextReviewAt == nil || pass.LastCreditAt == nil {
		t.Fatalf("pass = %+v", pass)
	}
	if want := Due(1, now, nil); !pass.NextReviewAt.Equal(want) {
		t.Fatalf("next = %v, want %v", pass.NextReviewAt, want)
	}
	// 이미 상자 2인 목표는 통과로 내려가지 않는다.
	next := now.AddDate(0, 0, 3)
	high := ApplyKnownVerify(Mastery{Box: 2, Best: BestUnaided, NextReviewAt: &next, Verify: true}, OutcomeUnaidedCorrect, now, nil)
	if high.Box != 2 || !high.NextReviewAt.Equal(next) || high.Verify {
		t.Fatalf("high = %+v", high)
	}
	fail := ApplyKnownVerify(fresh, OutcomeFailed, now, nil)
	if fail.Box != 0 || fail.Best != BestRevealed || fail.Verify || fail.LastFailAt == nil {
		t.Fatalf("fail = %+v", fail)
	}
	if held := ApplyKnownVerify(fresh, OutcomeHeld, now, nil); held != fresh {
		t.Fatalf("held must change nothing: %+v", held)
	}
	for outcome, want := range map[string]bool{OutcomeFailed: true, OutcomeAided: true, OutcomeUnaidedPartial: true,
		OutcomeUnaidedCorrect: false, OutcomeHeld: false, OutcomeRecorded: false} {
		if SuggestRelearn(outcome) != want {
			t.Errorf("SuggestRelearn(%s) != %v", outcome, want)
		}
	}
}

func TestMixKnownVerify(t *testing.T) {
	cases := []struct{ limit, due, verify, wantReviews, wantVerify int }{
		{10, 15, 5, 8, 2},
		{10, 3, 1, 3, 1},
		{10, 0, 2, 0, 2},
		{2, 5, 2, 1, 1},
		{1, 5, 2, 1, 0},
		{1, 0, 2, 0, 1},
		{0, 5, 2, 0, 0},
	}
	for _, c := range cases {
		r, v := MixKnownVerify(c.limit, c.due, c.verify)
		if r != c.wantReviews || v != c.wantVerify {
			t.Errorf("MixKnownVerify(%d,%d,%d) = %d,%d want %d,%d", c.limit, c.due, c.verify, r, v, c.wantReviews, c.wantVerify)
		}
	}
	sum := SummarizeReviews(ReviewCounts{Due: 3, Verify: 2}, 10, GraderSelf)
	if sum.Verify != 2 || sum.EstimatedMinutes != 5 {
		t.Fatalf("summary = %+v", sum)
	}
}

func TestInterleaveVerify(t *testing.T) {
	cases := map[string]string{
		strings.Join(InterleaveVerify([]string{"r1", "r2", "r3", "r4"}, []string{"v1", "v2"}), ","): "r1,v1,r2,r3,v2,r4",
		strings.Join(InterleaveVerify(nil, []string{"v1", "v2"}), ","):                              "v1,v2",
		strings.Join(InterleaveVerify([]string{"r1"}, []string{"v1"}), ","):                         "r1,v1",
		strings.Join(InterleaveVerify([]string{"r1", "r2"}, nil), ","):                              "r1,r2",
	}
	for got, want := range cases {
		if got != want {
			t.Errorf("interleave = %s, want %s", got, want)
		}
	}
}

func TestPickPlacement(t *testing.T) {
	c := func(id string, depth, order int, state string) PlacementConcept {
		return PlacementConcept{ConceptID: id, ObjectiveID: id + "#1", Depth: depth, Order: order, State: state}
	}
	got := PickPlacement([]PlacementConcept{c("hard", 2, 1, ""), c("easy", 1, 2, ""), c("mid", 1, 3, ""), c("done", 1, 0, Known)})
	ids := []string{}
	for _, p := range got {
		ids = append(ids, p.ConceptID)
	}
	if strings.Join(ids, ",") != "easy,mid,hard" {
		t.Fatalf("order = %v", ids)
	}
	if PickPlacement([]PlacementConcept{c("a", 1, 1, ""), c("b", 1, 2, Completed), c("c", 1, 3, "")}) != nil {
		t.Fatal("fewer than 3 open concepts → no placement")
	}
	many := []PlacementConcept{}
	for i := 0; i < 9; i++ {
		many = append(many, c(string(rune('a'+i)), 1, i, ""))
	}
	ids = ids[:0]
	for _, p := range PickPlacement(many) {
		ids = append(ids, p.ConceptID)
	}
	if strings.Join(ids, ",") != "a,c,e,g,i" {
		t.Fatalf("spread = %v", ids)
	}
}
