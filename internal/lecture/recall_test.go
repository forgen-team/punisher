package lecture

import (
	"encoding/json"
	"strings"
	"testing"
)

func recallLecture() string {
	return strings.Replace(strings.Replace(materialLecture(), "@@exercise\n0110", "@@exercise kind=guided\n0110", 1),
		"@@sources\n- [Integer types]", "@@recall objective=os.process#1\n방금 본 4비트 덧셈에서 부호 비트가 바뀐 이유는?\n떠올릴 것: 자리올림이 최상위 비트로 넘어갔다\n@@sources\n- [Integer types]", 1)
}

func TestRecallBlockParseRenderValidate(t *testing.T) {
	c := mustParse(t, recallLecture())
	recalls := RecallBlocks(c)
	if len(recalls) != 1 || recalls[0].Section != SectionPrinciples || recalls[0].Block.Objective != "os.process#1" {
		t.Fatalf("recall blocks: %+v", recalls)
	}
	q, key := RecallParts(recalls[0].Block.Text)
	if q != "방금 본 4비트 덧셈에서 부호 비트가 바뀐 이유는?" || key != "자리올림이 최상위 비트로 넘어갔다" {
		t.Fatalf("parts: %q / %q", q, key)
	}
	if ex := c.Sections[1].Blocks[2]; ex.Type != BlockExercise || ex.Kind != ExerciseGuided {
		t.Fatalf("exercise kind: %+v", ex)
	}
	if issues := Validate(c, ctxFor()); len(issues) != 0 {
		t.Fatalf("valid lecture: %v", issues)
	}
	again := mustParse(t, RenderMarkup(c))
	a, _ := json.Marshal(c)
	b, _ := json.Marshal(again)
	if string(a) != string(b) {
		t.Fatalf("render round trip changed the content:\n%s\n%s", a, b)
	}
	if m := MeasureContent(c); m.Recalls != 1 {
		t.Fatalf("recalls measured: %+v", m)
	}
}

func TestRecallAndExerciseKindRules(t *testing.T) {
	noKey := strings.Replace(recallLecture(), "떠올릴 것: 자리올림이 최상위 비트로 넘어갔다\n", "", 1)
	if !hasIssue(Validate(mustParse(t, noKey), ctxFor()), "떠올릴 것") {
		t.Fatal("recall without key must fail")
	}
	badKind := strings.Replace(recallLecture(), "kind=guided", "kind=fill", 1)
	if !hasIssue(Validate(mustParse(t, badKind), ctxFor()), "exercise kind") {
		t.Fatal("unknown exercise kind must fail")
	}
}

// Old stored lectures have no objective/kind fields: their JSON (and hash) is unchanged.
func TestOldLectureJSONUnchanged(t *testing.T) {
	c := mustParse(t, materialLecture())
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), `"objective"`) {
		t.Fatalf("empty objective must be omitted: %s", raw)
	}
}
