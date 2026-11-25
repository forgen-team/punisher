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
		"@@diff lang=", "@@chart", "7,000자", "@@recall objective=", "kind=modify", "MDN이 아닌", "대상 개념 자신"} {
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

// TestReviewGateAndRevisionRules: the review JSON carries a gate, the
// unverifiable-claim rule, depth-1 scope and the partial rewrite contract.
func TestReviewGateAndRevisionRules(t *testing.T) {
	for _, want := range []string{`"gate": "blocking|advisory"`, "조건 없이 단정했고 틀렸을 근거를 댈 수 있을 때만 blocking", "advisory 만 있으면 pass", "처방 미해결", "에러·출력 불일치"} {
		if !strings.Contains(ReviewTask, want) {
			t.Errorf("review task lacks %q", want)
		}
	}
	for _, want := range []string{"언어 표준만으로 결정되는 코드", "\"확인할 것\""} {
		if !strings.Contains(FactualRules, want) {
			t.Errorf("factual rules lack %q", want)
		}
	}
	depth1 := LectureContext{ConceptID: "db.relational", Depth: 1}
	depth2 := LectureContext{ConceptID: "db.relational", Depth: 2}
	for name, p := range map[string]string{
		"outline": Outline(depth1, Defaults()), "lecture": Lecture(depth1, "", "", Defaults()),
		"review": Review(depth1, "@@lecture", Defaults()), "revision": LectureRevision(depth1, "", "@@lecture", "- 지적", Defaults()),
	} {
		if !strings.Contains(p, "## 깊이 1 범위") || !strings.Contains(p, "## 사실 정확성 규칙") {
			t.Errorf("%s prompt lacks the depth-1 scope or factual rules", name)
		}
	}
	if strings.Contains(Lecture(depth2, "", "", Defaults()), "## 깊이 1 범위") {
		t.Error("depth 2 lectures have no depth-1 scope")
	}
	rev := LectureRevision(depth2, "", "@@lecture\ntitle: 이전", "- [major/factual_error] 틀림", NewSet(map[string]string{BlockLectureFormat: "형식"}))
	for _, want := range []string{"<<<DRAFT\n@@lecture\ntitle: 이전\nDRAFT>>>", "틀림", "글자 하나 바꾸지 말고", "그 주장을 뺍니다", "형식"} {
		if !strings.Contains(rev, want) {
			t.Errorf("revision prompt lacks %q", want)
		}
	}
}

// TestRecheckAndAdjudicationPrompts: the blocking evidence rule (confidence,
// reader impact, no blocking for unverifiable compatibility or conditioned
// sentences) reaches every review prompt, even with an edited review block;
// the recheck carries the findings and changed blocks only; the adjudication
// asks one question about one finding.
func TestRecheckAndAdjudicationPrompts(t *testing.T) {
	c := LectureContext{ConceptID: "db.relational", ConceptName: "관계형 모델", Depth: 1}
	edited := NewSet(map[string]string{BlockReviewTask: "내 감수", BlockReviewRecheck: "내 재검수"})
	recheck := Recheck(c, []RecheckFinding{{ID: 1, Kind: "typo", Section: "walkthrough", Quote: "INSERT", Message: "id 없음",
		Before: []string{"@@code lang=sql\nINSERT"}, After: []string{"@@code lang=sql\nINSERT id"}}},
		[]RecheckBlock{{Section: "walkthrough", Text: "@@code lang=sql\nINSERT id"}}, edited)
	for name, p := range map[string]string{"review": Review(c, "@@lecture", edited), "recheck": recheck} {
		for _, want := range []string{"## blocking 지적의 근거", "\"reader_impact\"", "0.7 미만", "렌더러 호환", "이미 조건", "## 사실 정확성 규칙"} {
			if !strings.Contains(p, want) {
				t.Errorf("%s prompt lacks %q", name, want)
			}
		}
	}
	for _, want := range []string{"### 지적 1 [typo · walkthrough]", "수정 전 블록", "INSERT id", "### 바뀐 블록 1 (walkthrough)", "내 재검수"} {
		if !strings.Contains(recheck, want) {
			t.Errorf("recheck lacks %q", want)
		}
	}
	if strings.Contains(recheck, "<<<LECTURE") || !strings.Contains(RecheckTask, `"resolved": true`) || !strings.Contains(RecheckTask, "new_issues") {
		t.Error("the recheck reads blocks, not the lecture, and answers resolved/new_issues")
	}
	adj := Adjudication(c, AdjudicationCase{Kind: "factual_error", Section: "principles", Quote: "기본으로 꺼져 있다",
		Message: "빌드 옵션", ReaderImpact: "켜져 있다고 믿는다", Blocks: []string{"@@markdown\n기본으로 꺼져 있다"}})
	for _, want := range []string{"기본으로 꺼져 있다", "켜져 있다고 믿는다", "실제로 틀렸거나 독자를 오도", "## 사실 정확성 규칙"} {
		if !strings.Contains(adj, want) {
			t.Errorf("adjudication lacks %q", want)
		}
	}
	if !strings.Contains(SystemAdjudicator, `"decision": "uphold" | "dismiss"`) || strings.Contains(SystemAdjudicator, "기술 감수자") {
		t.Error("adjudicator answers uphold|dismiss and is not the reviewer")
	}
}
