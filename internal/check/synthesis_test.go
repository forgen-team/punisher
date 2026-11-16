package check

import (
	"reflect"
	"strings"
	"testing"
)

func synthItem(covers ...string) Item {
	return Item{Format: FormatIntegrative, Prompt: "어디부터 보겠어요?",
		Rubric: Rubric{Must: []string{"DNS 지연을 의심"}, Answer: "먼저 dig 로 확인", Covers: covers}}
}

func TestValidateSynthesis(t *testing.T) {
	sc := SynthesisContext{ObjectiveIDs: []string{"a#1", "b#1", "c#1"}, ConceptOf: map[string]string{"a#1": "a", "b#1": "b", "c#1": "c"}}
	ok := ItemSet{Items: []Item{synthItem("a#1", "b#1"), synthItem("b#1", "c#1")}}
	if p := ValidateSynthesis(ok, sc); len(p) != 0 {
		t.Fatalf("valid set rejected: %v", p)
	}
	bad := ItemSet{Items: []Item{synthItem("a#1"), synthItem("a#1", "x#9")}}
	p := strings.Join(ValidateSynthesis(bad, sc), "\n")
	for _, want := range []string{"서로 다른 개념 2개", "고른 목표가 아닙니다"} {
		if !strings.Contains(p, want) {
			t.Errorf("problems miss %q:\n%s", want, p)
		}
	}
	one := ItemSet{Items: []Item{synthItem("a#1", "b#1")}}
	if p := ValidateSynthesis(one, sc); len(p) == 0 {
		t.Fatal("one item must be rejected")
	}
	withObjective := synthItem("a#1", "b#1")
	withObjective.ObjectiveID, withObjective.Format = "a#1", FormatApply
	if p := ValidateSynthesis(ItemSet{Items: []Item{withObjective, synthItem("a#1", "c#1")}}, sc); len(p) != 0 {
		t.Fatalf("objective_id/format are set by code, not rejected: %v", p)
	}
	if n := NormalizeSynthesis(ItemSet{Items: []Item{withObjective}}, sc.ObjectiveIDs)[0]; n.ObjectiveID != "" || n.Format != FormatIntegrative {
		t.Fatalf("normalized = %+v", n)
	}
}

func TestNormalizeSynthesis(t *testing.T) {
	got := NormalizeSynthesis(ItemSet{Items: []Item{synthItem("c#1", "a#1", "a#1")}}, []string{"a#1", "b#1", "c#1"})
	if got[0].Purpose != PurposeSynthesis || !reflect.DeepEqual(got[0].Rubric.Covers, []string{"a#1", "c#1"}) || string(got[0].Material) != "{}" {
		t.Fatalf("normalized = %+v", got[0])
	}
}
