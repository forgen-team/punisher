package services

import (
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/prompts"
)

func personalTarget(owner string) Target {
	pc := prompts.LectureContext{
		ConceptID: "t", ConceptName: "대상", Area: "os", Depth: 1,
		Learned: []prompts.ConceptRef{{ID: "p", Name: "약한 선행"}, {ID: "q", Name: "튼튼한 선행"}},
		Catalog: []prompts.CatalogRef{{ID: "r", Name: "약한 다른 개념", Area: "db", Learned: true}, {ID: "s", Name: "안 배운 개념", Area: "db"},
			{ID: "u", Name: "튼튼한 다른 개념", Area: "db", Learned: true}},
		StyleHint: prompts.StyleHintFor("too_hard"),
	}
	return Target{Prompt: pc, Owner: owner, personal: personalContext{
		weak:           map[string]bool{"p": true, "r": true, "s": true},
		misconceptions: []prompts.Misconception{{ConceptName: "약한 선행", Text: "락은 항상 행 단위다"}},
	}}
}

func refIDs(refs []prompts.ConceptRef) string {
	ids := []string{}
	for _, r := range refs {
		ids = append(ids, r.ID)
	}
	return strings.Join(ids, ",")
}

// M6: user-owned generations split weak concepts out of "배운 개념" and Catalog ✓
// into "다시 짚을 개념" and add the misconceptions; library lectures get none of it.
func TestLectureContextPersonalGate(t *testing.T) {
	lib := personalTarget("").LectureContext()
	if refIDs(lib.Learned) != "p,q" || len(lib.Revisit) != 0 || len(lib.Misconceptions) != 0 || len(lib.Catalog) != 3 {
		t.Fatalf("library context must stay shared: %+v", lib)
	}
	libText := prompts.Lecture(lib, "", "", prompts.Defaults())
	if strings.Contains(libText, "다시 짚을 개념") || strings.Contains(libText, "인정한 오개념") || strings.Contains(libText, "락은 항상") {
		t.Fatal("library prompt must not carry personal blocks")
	}

	own := personalTarget("u1").LectureContext()
	if refIDs(own.Learned) != "q" || refIDs(own.Revisit) != "p,r" {
		t.Fatalf("learned=%s revisit=%s", refIDs(own.Learned), refIDs(own.Revisit))
	}
	for _, c := range own.Catalog {
		if c.ID == "r" {
			t.Fatal("a weak learned concept leaves the catalog (one list only)")
		}
	}
	if len(own.Catalog) != 2 {
		t.Fatalf("unlearned and strong catalog entries stay: %+v", own.Catalog)
	}
	text := prompts.Lecture(own, "", "", prompts.Defaults())
	style := strings.Index(text, "## 설명 방식 조정")
	block := strings.Index(text, "## 이 수강생이 인정한 오개념")
	if !strings.Contains(text, "## 다시 짚을 개념") || !strings.Contains(text, "- 약한 선행: 락은 항상 행 단위다") || style < 0 || block < style {
		t.Fatalf("personal blocks missing or out of place:\n%s", text)
	}
	if strings.Contains(text, "약한 다른 개념✓") {
		t.Fatal("weak concept must not be marked ✓")
	}
}

func TestSplitLearnedKeepsOneList(t *testing.T) {
	kept, revisit := splitLearned([]prompts.ConceptRef{{ID: "a"}, {ID: "b"}}, map[string]bool{"b": true})
	if refIDs(kept) != "a" || refIDs(revisit) != "b" {
		t.Fatalf("kept=%s revisit=%s", refIDs(kept), refIDs(revisit))
	}
	if kept, revisit = splitLearned(nil, nil); kept == nil || len(revisit) != 0 {
		t.Fatal("empty input → empty kept list")
	}
}
