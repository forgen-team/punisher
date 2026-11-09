package prompts

import (
	"regexp"
	"strings"
	"testing"
)

func TestRegistryCoversEditableBlocks(t *testing.T) {
	want := []string{BlockOutlineTask, BlockTeacherSystem, BlockLectureFormat, BlockReviewerSystem, BlockReviewTask,
		BlockGlossSystem, BlockQASystem, BlockExplainSystem, BlockResolveSystem}
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
