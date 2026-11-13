package check

import (
	"encoding/json"
	"strings"
	"testing"
)

const lectureSQL = "SELECT 2147483647::int + 1;\n-- ERROR: integer out of range"

// TestPredictCopyRejected (design v2.2 §4 predict 출처): a predict item whose
// material.code is a lecture code block (whitespace/case aside) is rejected;
// new code and non-predict formats are not.
func TestPredictCopyRejected(t *testing.T) {
	withCode := func(format, code string) ItemSet {
		set := goodSet()
		raw, _ := json.Marshal(map[string]string{"code": code, "lang": "sql"})
		set.Items[0].Format, set.Items[0].Material = format, raw
		return set
	}
	ctx := sc
	ctx.LectureCode = []string{lectureSQL, "ls -al"}
	for _, code := range []string{lectureSQL, "select 2147483647::INT + 1;\n\n--   error: integer out of range  "} {
		if p := ValidateItems(withCode(FormatPredict, code), ctx); !strings.Contains(strings.Join(p, "\n"), "강의의 코드 블록과 같습니다") {
			t.Errorf("copy %q must be rejected: %v", code, p)
		}
	}
	if p := ValidateItems(withCode(FormatPredict, "SELECT 32767::smallint + 1;"), ctx); len(p) != 0 {
		t.Fatalf("new code is fine: %v", p)
	}
	if p := ValidateItems(withCode(FormatApply, lectureSQL), ctx); len(p) != 0 {
		t.Fatalf("only predict items are checked: %v", p)
	}
	// a long excerpt of a lecture block is a copy; a short common line is not.
	if !CopiesLectureCode(json.RawMessage(`{"code":"SELECT 2147483647::int + 1;\n-- ERROR: integer out"}`), ctx.LectureCode) {
		t.Fatal("a 40+ rune excerpt is a copy")
	}
	if CopiesLectureCode(json.RawMessage(`{"code":"ls"}`), ctx.LectureCode) || CopiesLectureCode(json.RawMessage(`{}`), ctx.LectureCode) {
		t.Fatal("a short fragment or no code is not a copy")
	}
}

// TestVerdictConsistency (design v2.2 §4 판정 일관성): correct with met not
// covering every must becomes partial; the move rule then follows partial.
func TestVerdictConsistency(t *testing.T) {
	must := []string{"메모리에 남김", "디스크 접근 감소"}
	learner := "메모리에 남겨서 빨라져요"
	r := TutorReply{Verdict: "correct", Met: []string{"메모리에 남김"}, Missing: []string{"디스크 접근 감소"},
		Quote: "메모리", Confidence: 0.9, TutorMove: MoveNarrow, TutorText: "다시 읽을 때 무엇이 줄어들까요?"}
	got, p := ValidateTutorReply(r, Stage{Answers: 1}, must, answer, learner)
	if got.Verdict != "partial" || len(p) != 0 {
		t.Fatalf("correct without every must = %s %v, want partial with no problem", got.Verdict, p)
	}
	// the closing move of a correct grade no longer fits: rewrite asked, with the reason.
	r.TutorMove, r.TutorText = MoveNext, "맞아요."
	if _, p := ValidateTutorReply(r, Stage{Answers: 1}, must, answer, learner); !strings.Contains(strings.Join(p, " "), "partial 입니다") {
		t.Fatalf("problems = %v", p)
	}
	// at the last answer the closing move is the same — corrected silently.
	if got, p := ValidateTutorReply(r, Stage{Answers: 3}, must, answer, learner); got.Verdict != "partial" || len(p) != 0 {
		t.Fatalf("last answer = %s %v", got.Verdict, p)
	}
	// met copies must (spacing aside) → stays correct.
	r.Met = []string{"메모리에  남김", "디스크 접근 감소(재읽기)"}
	if got, _ := ValidateTutorReply(r, Stage{Answers: 1}, must, answer, learner); got.Verdict != "correct" {
		t.Fatalf("covered must = %s", got.Verdict)
	}
	reg, _ := ValidateRegrade(RegradeReply{Verdict: "correct", Met: []string{"메모리에 남김"}, Quote: "메모리", Confidence: 0.9}, must, learner)
	if reg.Verdict != "partial" || DisputeAccepted(reg) {
		t.Fatalf("regrade without every must = %+v", reg)
	}
}
