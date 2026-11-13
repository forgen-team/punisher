package prompts

import (
	"regexp"
	"strings"
	"testing"
)

func TestRegistryCoversEditableBlocks(t *testing.T) {
	want := []string{BlockOutlineTask, BlockTeacherSystem, BlockLectureFormat, BlockReviewerSystem, BlockReviewTask,
		BlockGlossSystem, BlockQASystem, BlockExplainSystem, BlockResolveSystem, BlockCheckItems, BlockCheckTutor, BlockCheckPretest}
	for _, id := range want {
		b, ok := LookupBlock(id)
		if !ok || strings.TrimSpace(b.Default) == "" || b.Purpose == "" || b.StageTitle == "" {
			t.Errorf("block %s missing or incomplete: %+v", id, b)
		}
	}
	if b, _ := LookupBlock(BlockLectureFormat); b.Warning == "" {
		t.Error("format block must carry a warning")
	}
	if b, _ := LookupBlock(BlockResolveSystem); !b.Advanced {
		t.Error("resolver must be folded as advanced")
	}
}

func TestOverridePrecedence(t *testing.T) {
	def := Defaults()
	if def.Raw(BlockTeacherSystem) != SystemTeacher || def.Overridden(BlockTeacherSystem) {
		t.Fatal("defaults must be the code constants")
	}
	set := NewSet(map[string]string{BlockTeacherSystem: "내 강사 원칙 {{concept_name}} {{unknown}}", "nope": "x"})
	if !set.Overridden(BlockTeacherSystem) || set.Overridden("nope") || set.Overridden(BlockQASystem) {
		t.Fatal("only registered overridden blocks count")
	}
	got := set.Text(BlockTeacherSystem, map[string]string{"concept_name": "페이지 캐시", "unknown": "z"})
	if got != "내 강사 원칙 페이지 캐시 {{unknown}}" {
		t.Fatalf("text = %q (declared vars substituted, undeclared kept)", got)
	}
	if set.Raw(BlockQASystem) != SystemQA {
		t.Fatal("non-overridden block must fall back to the default")
	}
}

func TestBuildersUseTheSet(t *testing.T) {
	c := LectureContext{ConceptID: "os.page-cache", ConceptName: "페이지 캐시", Depth: 2}
	set := NewSet(map[string]string{
		BlockLectureFormat: "## 내 형식 {{depth}}", BlockOutlineTask: "## 내 개요 {{concept_name}}", BlockReviewTask: "## 내 감수",
	})
	if l := Lecture(c, "", "", set); !strings.Contains(l, "## 내 형식 2") || strings.Contains(l, "@@lecture") {
		t.Fatalf("lecture prompt ignores override: %s", l)
	}
	if o := Outline(c, set); !strings.Contains(o, "## 내 개요 페이지 캐시") || strings.Contains(o, "situation_question (약 3분)") {
		t.Fatal("outline prompt ignores override")
	}
	if r := Review(c, "x", set); !strings.Contains(r, "## 내 감수") {
		t.Fatal("review prompt ignores override")
	}
	if l := Lecture(c, "", "", Defaults()); !strings.Contains(l, MarkupSpec) {
		t.Fatal("default lecture prompt must contain MarkupSpec")
	}
}

var userVersionRe = regexp.MustCompile(`^` + regexp.QuoteMeta(Version) + `\+u:[0-9a-f]{8}$`)

func TestVersionHash(t *testing.T) {
	if v := Defaults().Version(FullBlocks...); v != Version {
		t.Fatalf("default version = %q", v)
	}
	a := NewSet(map[string]string{BlockTeacherSystem: "A"})
	b := NewSet(map[string]string{BlockTeacherSystem: "B"})
	va, vb := a.Version(InstantBlocks...), b.Version(InstantBlocks...)
	if !userVersionRe.MatchString(va) || va == vb {
		t.Fatalf("versions %q %q", va, vb)
	}
	if va != NewSet(map[string]string{BlockTeacherSystem: "A"}).Version(BlockLectureFormat, BlockTeacherSystem) {
		t.Fatal("version must not depend on block order")
	}
	// An override of a block the generation does not use leaves the version alone.
	if v := NewSet(map[string]string{BlockQASystem: "x"}).Version(InstantBlocks...); v != Version {
		t.Fatalf("unused override changed version: %q", v)
	}
	if h := a.Hash(BlockTeacherSystem); len(h) != 8 || Defaults().Hash(BlockTeacherSystem) != "" {
		t.Fatalf("hash = %q", h)
	}
}

func TestUnknownVars(t *testing.T) {
	got := UnknownVars("{{concept_name}} {{ foo }} {{foo}} {{bar}} {{", []Var{{Name: "concept_name"}})
	if strings.Join(got, ",") != "foo,bar" {
		t.Fatalf("unknown = %v", got)
	}
}

func TestAssumedKnowledgeRules(t *testing.T) {
	if Version != "punisher-prompts-2026-11-10.1" {
		t.Fatalf("version = %s (bump when default blocks change)", Version)
	}
	for _, want := range []string{"전제하지 않습니다", "첫 등장에서 정의합니다", "2~3문장", "\"당연히\""} {
		if !strings.Contains(SystemTeacher, want) {
			t.Errorf("teacher lacks %q", want)
		}
	}
	for _, want := range []string{"(term:os.process)", "문장 중간에 \"@@term\"을 쓰지 않습니다"} {
		if !strings.Contains(MarkupSpec, want) {
			t.Errorf("markup spec lacks %q", want)
		}
	}
	for _, want := range []string{"초보 독자 시뮬레이션", "assumed_knowledge", "\"reader_gaps\"", "INSERT INSERT INTO"} {
		if !strings.Contains(ReviewTask, want) {
			t.Errorf("review task lacks %q", want)
		}
	}
	if !strings.Contains(OutlineTask, "terms_to_define") {
		t.Error("outline lacks terms_to_define")
	}
	p := Lecture(LectureContext{ConceptID: "os.process", Unlearned: []ConceptRef{{ID: "os.kernel-and-syscall", Name: "커널"}}}, "", "", Defaults())
	if !strings.Contains(p, "## 아직 배우지 않은 선행 개념") || !strings.Contains(p, "os.kernel-and-syscall: 커널") {
		t.Fatalf("lecture prompt lacks unlearned prerequisites: %s", p)
	}
}

func TestCatalogListedForLinking(t *testing.T) {
	p := Lecture(LectureContext{Catalog: []CatalogRef{
		{ID: "net.nat", Name: "NAT", Area: "network"}, {ID: "net.subnet", Name: "서브넷", Area: "network", Learned: true},
		{ID: "os.signal", Name: "시그널", Area: "os"},
	}}, "", "", Defaults())
	if !strings.Contains(p, "- network: net.nat NAT, net.subnet 서브넷✓\n- os: os.signal 시그널\n") {
		t.Fatalf("catalog block wrong:\n%s", p)
	}
}

// 2026-11-09: 즉석 강의에서 @@markdown 안 mermaid 펜스가 그림 대신 코드로 보였다.
func TestMarkupSpecKeepsDiagramsInMermaidBlocks(t *testing.T) {
	for _, want := range []string{"반드시 따로 @@mermaid 블록", "코드펜스 없이 도식 문법만"} {
		if !strings.Contains(MarkupSpec, want) {
			t.Errorf("markup spec lacks %q", want)
		}
	}
}

func TestCheckBlocksAndBuilders(t *testing.T) {
	for _, id := range []string{BlockCheckItems, BlockCheckTutor} {
		b, _ := LookupBlock(id)
		if b.Stage != StageCheck || b.StageTitle != "확인" || b.Warning == "" {
			t.Fatalf("%s: %+v", id, b)
		}
	}
	items := CheckItems(CheckItemsContext{ConceptName: "페이지 캐시", Objectives: []CheckObjective{{ID: "c#1", Text: "읽기 경로"}},
		Anchors: []string{"s1"}, Sections: []string{"[s1] 본문"}, Revisions: []string{"must 가 비었습니다"}})
	for _, want := range []string{"c#1: 읽기 경로", "purpose review", "s1", "must 가 비었습니다"} {
		if !strings.Contains(items, want) {
			t.Fatalf("items prompt lacks %q:\n%s", want, items)
		}
	}
	tutor := CheckTutor(CheckTutorContext{Prompt: "왜 빨라지나요?", Must: []string{"메모리"}, Answer: "모범",
		LearnerAnswer: "내 답", Allowed: []string{"narrow", "next"}, Dialogue: []CheckTurnRef{{Role: "tutor", Move: "ask", Text: "왜 빨라지나요?"}}})
	for _, want := range []string{"통합 문항", "narrow | next", "내 답", "tutor(ask)"} {
		if !strings.Contains(tutor, want) {
			t.Fatalf("tutor prompt lacks %q:\n%s", want, tutor)
		}
	}
	if r := CheckRegrade("문항", []string{"a"}, nil, "모범", "답"); !strings.Contains(r, "## 학습자 답\n답") || strings.Contains(r, "문답") {
		t.Fatalf("regrade prompt: %s", r)
	}
}

func TestPretestBlockAndBuilder(t *testing.T) {
	b, _ := LookupBlock(BlockCheckPretest)
	if b.Stage != StagePretest || b.Warning == "" || !strings.Contains(b.Default, "출력은 JSON 하나뿐") {
		t.Fatalf("pretest block: %+v", b)
	}
	objectives := []DiagnosticObjective{{ID: "c#1", ConceptName: "페이지 캐시", Text: "읽기 경로"}}
	pre := Diagnostic(DiagnosticContext{Purpose: "pretest", Objectives: objectives, Count: 2, Revisions: []string{"보기가 2개"}})
	for _, want := range []string{"c#1 (개념: 페이지 캐시): 읽기 경로", "정확히 2개", "보기가 2개"} {
		if !strings.Contains(pre, want) {
			t.Fatalf("pretest prompt lacks %q:\n%s", want, pre)
		}
	}
	if kv := Diagnostic(DiagnosticContext{Purpose: "known_verify", Objectives: objectives}); !strings.Contains(kv, "known_verify") || strings.Contains(kv, "%!") {
		t.Fatalf("known verify prompt: %s", kv)
	}
	if pl := Diagnostic(DiagnosticContext{Purpose: "placement", Objectives: objectives}); !strings.Contains(pl, "format choice") {
		t.Fatalf("placement prompt: %s", pl)
	}
}
