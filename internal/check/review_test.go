package check

import (
	"strings"
	"testing"
)

// TestReviewStageNarrowsOnce: a tutored review allows one narrowing question,
// then the item ends (no explain / reask), unlike a check.
func TestReviewStageNarrowsOnce(t *testing.T) {
	first := Stage{Answers: 1, Kind: PurposeReview}
	if got := strings.Join(first.AllowedMoves(), "|"); got != "narrow|next" {
		t.Fatalf("first answer moves = %s", got)
	}
	if first.ExpectedMove("partial") != MoveNarrow || first.ExpectedMove("correct") != MoveNext {
		t.Fatal("first answer: partial narrows, correct moves on")
	}
	second := Stage{Answers: 2, Kind: PurposeReview, Last: true}
	if got := strings.Join(second.AllowedMoves(), "|"); got != "wrap" || second.ExpectedMove("incorrect") != MoveWrap {
		t.Fatalf("after narrowing the review item ends: %s", got)
	}
	if (Stage{Answers: 2, Kind: PurposeCheck}).ExpectedMove("incorrect") != MoveExplain {
		t.Fatal("a check still explains after the narrowed answer")
	}
	if MaxAnswersFor(PurposeReview) != 2 || MaxAnswersFor(PurposeCheck) != MaxAnswers || MaxAnswersFor("") != MaxAnswers {
		t.Fatal("answer limits by kind")
	}
}

func TestValidateVariants(t *testing.T) {
	ok := ItemSet{Items: []Item{goodItem("c#1", PurposeReview)}}
	if p := ValidateVariants(ok, sc, []string{"c#1"}); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	cases := map[string]ItemSet{
		"check purpose":   {Items: []Item{goodItem("c#1", PurposeCheck)}},
		"other objective": {Items: []Item{goodItem("c#2", PurposeReview)}},
		"two variants":    {Items: []Item{goodItem("c#1", PurposeReview), goodItem("c#1", PurposeReview)}},
		"integrative":     {Items: []Item{goodItem("", PurposeCheck)}},
		"empty":           {},
	}
	for name, set := range cases {
		if p := ValidateVariants(set, sc, []string{"c#1"}); len(p) == 0 {
			t.Errorf("%s: want problems", name)
		}
	}
}
