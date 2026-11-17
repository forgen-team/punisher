package learning

import (
	"reflect"
	"testing"
)

func TestFlagSections(t *testing.T) {
	got := FlagSections([]SectionSignal{
		{Section: "situation_question"},                                    // nothing
		{Section: "principles", Exercises: 3, ExerciseMs: 2},               // 2/3 missed
		{Section: "walkthrough", Exercises: 3, ExerciseMs: 1},              // 1/3: not enough
		{Section: "judgment_next", Labs: 1, LabMisses: 1, Questions: 2},    // lab + questions
		{Section: "x", Exercises: 1, ExerciseMs: 1, Labs: 1, LabMisses: 0}, // 1/2 → flagged, exercise reason only
		{Section: "y", Questions: 1},                                       // one question
	}, "")
	want := []SectionFlag{
		{Section: "principles", Reasons: []string{"직접 해 보기 3번 중 2번 틀리거나 건너뜀"}},
		{Section: "judgment_next", Reasons: []string{"실습 1번 중 1번 결과가 예상과 다름", "질문 2개"}},
		{Section: "x", Reasons: []string{"직접 해 보기 1번 중 1번 틀리거나 건너뜀"}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flags = %+v\nwant %+v", got, want)
	}
	if got := FlagSections(nil, ""); got == nil || len(got) != 0 {
		t.Fatalf("no signals must give an empty, non-nil list: %#v", got)
	}
}

func TestSectionOfAnchor(t *testing.T) {
	for in, want := range map[string]string{
		"principles-3": "principles", "situation_question-1": "situation_question", "header": "header", "": "",
	} {
		if got := SectionOfAnchor(in); got != want {
			t.Errorf("SectionOfAnchor(%q) = %q, want %q", in, got, want)
		}
	}
	if !AttemptMiss(ExerciseSkipped) || !AttemptMiss(LabUnreadable) || AttemptMiss(ExercisePartial) || AttemptMiss(LabMatched) {
		t.Fatal("miss verdicts")
	}
}

func TestFlagSectionsEnglish(t *testing.T) {
	got := FlagSections([]SectionSignal{{Section: "p", Exercises: 3, ExerciseMs: 2, Labs: 1, LabMisses: 1, Questions: 2}}, LocaleEN)
	want := []string{"missed or skipped 2 of 3 exercises", "1 of 1 labs differed from the expected result", "2 questions"}
	if len(got) != 1 || !reflect.DeepEqual(got[0].Reasons, want) {
		t.Fatalf("en flags = %+v", got)
	}
}
