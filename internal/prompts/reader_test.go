package prompts

import (
	"strings"
	"testing"
)

// The reader assumption every prompt had before levels existed (SystemTeacher
// opening sentence and rule 9 of punisher-prompts-2026-11-16.1).
const (
	oldTeacherReader  = "수강생은 실무 경력이 있는 개발자지만 CS·운영 기반 지식(OS, 네트워크, DB, 분산 시스템, 인프라, 보안)에 공백이 있습니다."
	oldTeacherAssumes = "수강생이 안다고 가정해도 되는 것은 웹 FE/BE 실무 범위(HTTP 요청·응답과 상태 코드, JSON, JS/TS 코드, SQL 기본 CRUD, git, cd·ls·cat·grep 정도의 셸)뿐입니다. OS·네트워크 스택·DB 내부·분산 시스템·인프라·보안의 용어, 약어, 도구 이름(예: systemd, 시그널, 서브넷, NAT, SYN, 격리 수준, 락, 시퀀스, SLO)은 모른다고 가정합니다."
)

func TestReaderProfilePractitionerKeepsOldAssumption(t *testing.T) {
	text := ReaderProfile(ReaderPractitioner, "ko")
	for _, want := range []string{oldTeacherReader, oldTeacherAssumes} {
		if !strings.Contains(text, want) {
			t.Errorf("practitioner section lacks the old assumption %q", want)
		}
	}
	if ReaderProfile("", "ko") != text || ReaderProfile("expert", "ko") != text {
		t.Error(`"" and unknown levels must be practitioner`)
	}
	for _, block := range []string{SystemTeacher, ReviewTask, SystemExplain, SystemQA, SystemCheckItems} {
		if strings.Contains(block, "웹 FE/BE") || strings.Contains(block, "경력 개발자") || strings.Contains(block, "실무 경력이 있는") {
			t.Errorf("editable default still fixes the reader: %.60q…", block)
		}
	}
}

func TestReaderProfileLevelsAndLocales(t *testing.T) {
	seen := map[string]bool{}
	for _, level := range []string{ReaderBeginner, ReaderNovice, ReaderPractitioner, ReaderAdvanced} {
		ko, en := ReaderProfile(level, "ko"), ReaderProfile(level, "en")
		for _, want := range []string{"## 독자 수준", "가르치는 사실과 학습 목표는 수준과 무관하게 같습니다", "설명 밀도", "우선합니다", "독자 시뮬레이션"} {
			if !strings.Contains(ko, want) {
				t.Errorf("%s ko lacks %q", level, want)
			}
		}
		for _, want := range []string{"## Reader level", "same at every level", "only the explanation density changes", "overrides any reader assumption", "reader simulations"} {
			if !strings.Contains(en, want) {
				t.Errorf("%s en lacks %q", level, want)
			}
		}
		if strings.Contains(en, "수강생") || seen[ko] || seen[en] {
			t.Errorf("%s: en must be English and each level's text distinct", level)
		}
		seen[ko], seen[en] = true, true
	}
}

// Every reader-facing builder appends the section of the given level, after the blocks.
func TestReaderProfileInEveryBuilder(t *testing.T) {
	r := Reader{Level: ReaderBeginner, Locale: "ko"}
	want := ReaderProfile(r.Level, r.Locale)
	lc := LectureContext{ConceptID: "os.x", ConceptName: "X", Depth: 1, Reader: r}
	qa := QAContext{ConceptName: "X", Reader: r}
	built := map[string]string{
		"outline":        Outline(lc, Defaults()),
		"lecture":        Lecture(lc, "", "", Defaults()),
		"revision":       LectureRevision(lc, "", "prev", "notes", Defaults()),
		"review":         Review(lc, "draft", Defaults()),
		"review legacy":  ReviewLegacy(lc, "draft", Defaults()),
		"recheck":        Recheck(lc, nil, nil, Defaults()),
		"explain":        ExplainSelection(ExplainContext{ConceptName: "X", Reader: r}, "sel"),
		"qa":             Question(qa, "q"),
		"alt":            AltExplanation(qa, ""),
		"inquiry":        Inquiry(InquiryContext{Question: "q", Reader: r}),
		"followup":       InquiryFollowup(FollowupContext{Question: "q", Reader: r}),
		"gloss":          Gloss("X", "os", "s", r),
		"check items":    CheckItems(CheckItemsContext{ConceptName: "X", Reader: r}),
		"check variants": CheckVariants(CheckItemsContext{ConceptName: "X", Reader: r}, nil),
		"check tutor":    CheckTutor(CheckTutorContext{ConceptName: "X", Reader: r}),
		"pretest":        Diagnostic(DiagnosticContext{Purpose: "pretest", Count: 1, Reader: r}),
	}
	for name, text := range built {
		if !strings.Contains(text, want) {
			t.Errorf("%s lacks the beginner reader section", name)
		}
	}
	review := built["review"]
	if strings.Index(review, want) < strings.Index(review, "1차 — 독자 시뮬레이션") {
		t.Error("the reader section must follow the review block it overrides")
	}
	if !strings.Contains(Lecture(LectureContext{}, "", "", Defaults()), oldTeacherAssumes) {
		t.Error("a zero LectureContext must still carry the practitioner assumption")
	}
}
