package prompts

import (
	"strings"
	"testing"
)

func TestLectureShapeByLevel(t *testing.T) {
	if s := LectureShape(ReaderBeginner, false); !strings.Contains(s, "구체 → 추상") || strings.Contains(s, "감수:") {
		t.Fatalf("beginner lecture shape: %q", s)
	}
	if s := LectureShape(ReaderNovice, true); !strings.Contains(s, "지연'으로 지적하지 않습니다") {
		t.Fatalf("novice review shape: %q", s)
	}
	if s := LectureShape("", true); !strings.Contains(s, "principles 첫머리") || strings.Contains(s, "지적하지 않습니다") {
		t.Fatalf("practitioner keeps definition first: %q", s)
	}
}

// The fixed sections reach the prompt even when the user edited the blocks around them.
func TestReviewFixedSectionsWithEditedBlocks(t *testing.T) {
	set := NewSet(map[string]string{BlockReviewTask: "내 감수 기준", BlockLectureFormat: "내 형식"})
	c := LectureContext{ConceptName: "x", Reader: Reader{Level: ReaderBeginner}}
	got := ReviewChecked(c, "@@lecture\n@@end", set, ReviewEvidence{SQL: []string{"walkthrough-2: 실행하면 오류"}})
	for _, want := range []string{"내 감수 기준", "\"rubric\"", "E13", "앱이 계산한 근거", "walkthrough-2: 실행하면 오류", "구체 → 추상"} {
		if !strings.Contains(got, want) {
			t.Errorf("review lacks %q", want)
		}
	}
	if strings.Contains(Review(c, "x", set), "앱이 계산한 근거") {
		t.Error("no evidence, no evidence section")
	}
	if l := Lecture(c, "", "", set); !strings.Contains(l, "내 형식") || !strings.Contains(l, "설명 순서와 연습") {
		t.Errorf("lecture prompt: %q", l)
	}
}

func TestRecomputePrompt(t *testing.T) {
	got := Recompute("정수", []RecomputeItem{{ID: "principles-3", Kind: "exercise", Question: "0110 + 0011 은?", Context: "4비트"}})
	if !strings.Contains(got, "### principles-3 (exercise)") || !strings.Contains(got, "CONTEXT") {
		t.Fatalf("recompute prompt: %q", got)
	}
	if !strings.Contains(SystemRecompute, `"answers"`) {
		t.Fatal("recompute output contract")
	}
}

func TestPretestWithLecture(t *testing.T) {
	dc := DiagnosticContext{Purpose: "pretest", Count: 2, Objectives: []DiagnosticObjective{{ID: "a#1", Text: "t"}},
		Lecture: []string{"### [principles] 원리\n[principles-1] 본문"}, Anchors: []string{"principles", "principles-1"}}
	got := Diagnostic(dc)
	if !strings.Contains(got, "rubric.anchor") || !strings.Contains(got, "[principles-1] 본문") {
		t.Fatalf("pretest with lecture: %q", got)
	}
	dc.Lecture = nil
	if strings.Contains(Diagnostic(dc), "rubric.anchor") {
		t.Fatal("no lecture, no anchor rule")
	}
}

func TestStyleHintsAreStructural(t *testing.T) {
	if h := StyleHintFor("too_hard"); !strings.Contains(h, "정의는 3개 이하") || !strings.Contains(h, "guided") {
		t.Fatalf("too_hard: %q", h)
	}
}
