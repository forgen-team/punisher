package check

import (
	"encoding/json"
	"testing"
)

func choiceItem(obj, purpose, prompt string, choices []string, correct int) Item {
	material, _ := json.Marshal(map[string]any{"choices": choices})
	return Item{ObjectiveID: obj, Purpose: purpose, Format: FormatChoice, Prompt: prompt, Material: material,
		Rubric: Rubric{Correct: &correct}}
}

func predictItem(obj string) Item {
	return Item{ObjectiveID: obj, Purpose: PurposePretest, Format: FormatPredict,
		Prompt: "write 직후 전원이 꺼지면 데이터는 어떻게 될까요?", Rubric: Rubric{Answer: "아직 디스크에 쓰지 않은 더티 페이지는 사라질 수 있다"}}
}

var cacheChoices = []string{"디스크에서 다시 읽는다", "메모리의 페이지 캐시에서 읽는다", "CPU 레지스터에서 읽는다"}

func TestPretestCount(t *testing.T) {
	for objectives, want := range map[int]int{1: 2, 2: 2, 3: 3, 4: 3} {
		if got := PretestCount(objectives); got != want {
			t.Errorf("PretestCount(%d) = %d, want %d", objectives, got, want)
		}
	}
}

func TestValidateDiagnosticPretest(t *testing.T) {
	dc := DiagnosticContext{Purpose: PurposePretest, ObjectiveIDs: []string{"c#1", "c#2"}}
	ok := ItemSet{Items: []Item{choiceItem("c#1", PurposePretest, "같은 파일을 두 번째 읽으면 어디서 읽을까요?", cacheChoices, 1), predictItem("c#2")}}
	if p := ValidateDiagnostic(ok, dc); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	bad := map[string]ItemSet{
		"one item":        {Items: []Item{predictItem("c#1")}},
		"missing c#2":     {Items: []Item{predictItem("c#1"), predictItem("c#1")}},
		"other objective": {Items: []Item{predictItem("c#1"), predictItem("x#1")}},
		"recall format":   {Items: []Item{predictItem("c#1"), {ObjectiveID: "c#2", Purpose: PurposePretest, Format: FormatRecall, Prompt: "왜?", Rubric: Rubric{Answer: "a"}}}},
		"check purpose":   {Items: []Item{predictItem("c#1"), {ObjectiveID: "c#2", Purpose: PurposeCheck, Format: FormatPredict, Prompt: "왜?", Rubric: Rubric{Answer: "a"}}}},
	}
	for name, set := range bad {
		if p := ValidateDiagnostic(set, dc); len(p) == 0 {
			t.Errorf("%s: want problems", name)
		}
	}
	// 목표가 하나면 2문항이 같은 목표를 묻는다.
	one := DiagnosticContext{Purpose: PurposePretest, ObjectiveIDs: []string{"c#1"}}
	if p := ValidateDiagnostic(ItemSet{Items: []Item{predictItem("c#1"), predictItem("c#1")}}, one); len(p) != 0 {
		t.Fatalf("single objective: %v", p)
	}
}

func TestValidateChoice(t *testing.T) {
	dc := DiagnosticContext{Purpose: PurposePlacement, ObjectiveIDs: []string{"c#1"}}
	prompt := "같은 파일을 두 번째 읽으면 어디서 읽을까요?"
	if p := ValidateDiagnostic(ItemSet{Items: []Item{choiceItem("c#1", PurposePlacement, prompt, cacheChoices, 1)}}, dc); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	cases := map[string]Item{
		"two options":   choiceItem("c#1", PurposePlacement, prompt, cacheChoices[:2], 0),
		"five options":  choiceItem("c#1", PurposePlacement, prompt, []string{"a1", "b2", "c3", "d4", "e5"}, 0),
		"no correct":    {ObjectiveID: "c#1", Purpose: PurposePlacement, Format: FormatChoice, Prompt: prompt, Material: json.RawMessage(`{"choices":["가나","다라","마바"]}`)},
		"correct range": choiceItem("c#1", PurposePlacement, prompt, cacheChoices, 3),
		"duplicate":     choiceItem("c#1", PurposePlacement, prompt, []string{"디스크", "디스크", "메모리"}, 2),
		"leak in prompt": choiceItem("c#1", PurposePlacement, "메모리의 페이지 캐시에서 읽는다는 말이 맞을까요?",
			cacheChoices, 1),
		"longest gives away": choiceItem("c#1", PurposePlacement, prompt,
			[]string{"디스크", "메모리에 남은 페이지 캐시에서 다시 읽어 디스크 접근을 줄인다", "레지스터"}, 1),
		"predict format": predictItem("c#1"),
	}
	for name, it := range cases {
		it.Purpose = PurposePlacement
		if p := ValidateDiagnostic(ItemSet{Items: []Item{it}}, dc); len(p) == 0 {
			t.Errorf("%s: want problems", name)
		}
	}
}

func TestValidateDiagnosticKnownVerify(t *testing.T) {
	dc := DiagnosticContext{Purpose: PurposeKnownVerify, ObjectiveIDs: []string{"c#1", "c#2"}}
	item := func(obj string) Item {
		it := goodItem(obj, PurposeKnownVerify)
		it.Rubric.Anchor = "" // 강의 없이 만든 문항: 앵커가 없다.
		return it
	}
	if p := ValidateDiagnostic(ItemSet{Items: []Item{item("c#1"), item("c#2")}}, dc); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
	noMust := item("c#2")
	noMust.Rubric.Must = nil
	choice := choiceItem("c#2", PurposeKnownVerify, "어디서 읽을까요?", cacheChoices, 1)
	for name, set := range map[string]ItemSet{
		"twice":   {Items: []Item{item("c#1"), item("c#1"), item("c#2")}},
		"no must": {Items: []Item{item("c#1"), noMust}},
		"choice":  {Items: []Item{item("c#1"), choice}},
	} {
		if p := ValidateDiagnostic(set, dc); len(p) == 0 {
			t.Errorf("%s: want problems", name)
		}
	}
}

func TestNormalizeDiagnosticSetsChoiceAnswer(t *testing.T) {
	it := choiceItem("c#1", PurposePretest, " 어디서 읽을까요? ", []string{" 디스크 ", "메모리", "레지스터"}, 1)
	it.Rubric.Answer = "모델이 쓴 다른 답"
	out := NormalizeDiagnostic(ItemSet{Items: []Item{it, predictItem("c#2")}})
	if out[0].Rubric.Answer != "메모리" || out[0].Prompt != "어디서 읽을까요?" || Choices(out[0].Material)[0] != "디스크" {
		t.Fatalf("choice normalize: %+v %s", out[0].Rubric, out[0].Material)
	}
	if out[1].Rubric.Correct != nil || string(out[1].Material) != "{}" {
		t.Fatalf("predict normalize: %+v %s", out[1].Rubric, out[1].Material)
	}
}

func TestKnownVerifyTakesOneAnswer(t *testing.T) {
	st := Stage{Answers: 1, Kind: PurposeKnownVerify}
	if moves := st.AllowedMoves(); len(moves) != 1 || moves[0] != MoveNext || st.ExpectedMove("partial") != MoveNext {
		t.Fatalf("known verify moves = %v", moves)
	}
	if !Recorded(PurposePretest) || !Recorded(PurposePlacement) || Recorded(PurposeKnownVerify) || Recorded(PurposeCheck) {
		t.Fatal("recorded kinds")
	}
}
