package prompts

import (
	"strings"
	"testing"
)

// TestFactualRulesInEveryGenerationPrompt (quality design v2.2 §3.1): the fixed
// block is in every generation prompt, also when the user overrides the
// instruction blocks (it is not an editable block).
func TestFactualRulesInEveryGenerationPrompt(t *testing.T) {
	lc := LectureContext{ConceptID: "ds.complexity", ConceptName: "복잡도"}
	qc := QAContext{ConceptID: "ds.complexity", ConceptName: "복잡도"}
	ic := CheckItemsContext{ConceptName: "복잡도", Objectives: []CheckObjective{{ID: "ds.complexity#1", Text: "x"}}}
	overridden := NewSet(map[string]string{BlockOutlineTask: "개요", BlockLectureFormat: "형식", BlockReviewTask: "검수"})
	got := map[string]string{
		"outline":   Outline(lc, overridden),
		"lecture":   Lecture(lc, "", "", overridden),
		"review":    Review(lc, "@@lecture", overridden),
		"legacy":    ReviewLegacy(lc, "@@lecture", overridden),
		"items":     CheckItems(ic),
		"variants":  CheckVariants(ic, nil),
		"tutor":     CheckTutor(CheckTutorContext{ConceptName: "복잡도"}),
		"pretest":   Diagnostic(DiagnosticContext{Purpose: "pretest", Count: 2}),
		"qa":        Question(qc, "왜?"),
		"alt":       AltExplanation(qc, ""),
		"explain":   ExplainSelection(ExplainContext{}, "구절"),
		"inquiry":   Inquiry(InquiryContext{Question: "왜?"}),
		"follow-up": InquiryFollowup(FollowupContext{Question: "왜?"}),
	}
	for name, p := range got {
		if !strings.Contains(p, "## 사실 정확성 규칙") || !strings.Contains(p, "자리올림/오버플로") {
			t.Errorf("%s prompt lacks the factual-accuracy rules", name)
		}
	}
	if !strings.Contains(got["legacy"], "구조 블록 도입 전 강의") || strings.Contains(got["review"], "구조 블록 도입 전 강의") {
		t.Error("only the backfill review carries the legacy note")
	}
}

func TestLectureQualityRules(t *testing.T) {
	for _, want := range []string{"procedural_prereqs", "thesis_check", "key_definitions", "scene_numbers", "최대 2개"} {
		if !strings.Contains(OutlineTask, want) {
			t.Errorf("outline task lacks %q", want)
		}
	}
	for _, want := range []string{"@@keypoint kind=definition", "@@trace", "@@pause", "@@exercise", "@@answer", "@@summary",
		"@@diff lang=", "@@chart", "8,500자", "MDN이 아닌", "대상 개념 자신"} {
		if !strings.Contains(MarkupSpec, want) {
			t.Errorf("markup spec lacks %q", want)
		}
	}
	for _, want := range []string{"자리올림/오버플로", "처방 추적 누락", "structure", "단계별로 계산해 확인", "\"확인 필요\""} {
		if !strings.Contains(ReviewTask, want) {
			t.Errorf("review task lacks %q", want)
		}
	}
	for _, want := range []string{"target_check", "transfer", "강의의 코드 블록을 그대로"} {
		if !strings.Contains(SystemCheckItems, want) {
			t.Errorf("check items lacks %q", want)
		}
	}
	for _, want := range []string{"must 문장을 그대로 복사", "반례 질문", "탓하는 문장"} {
		if !strings.Contains(SystemCheckTutor, want) {
			t.Errorf("tutor lacks %q", want)
		}
	}
}
