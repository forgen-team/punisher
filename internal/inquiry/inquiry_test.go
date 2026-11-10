package inquiry

import (
	"strings"
	"testing"

	"github.com/wooo-jin/punisher/internal/lecture"
)

const validMarkup = `@@lecture
title: 서버는 살아 있는데 CORS 오류가 나는 이유
concepts: db.connection-pool, net.cors, unknown.id
new_concept: infra.probe-design | 헬스체크 설계 | infra-ops | net.cors | 무엇을 확인하는 헬스체크인가
new_concept: BAD SLUG | 이름 | infra-ops
new_concept: infra.x-y | 이름 | not-an-area
@@section summary title=한 줄 답
@@markdown
[커넥션 풀](term:db.connection-pool)이 고갈되면 요청이 대기합니다. [헬스체크](term:infra.probe-design)는 DB를 보지 않습니다. [없는 것](term:nope.nope)
@@section causes title=원인 사슬
@@markdown
1. 풀 고갈 → 2. 대기·타임아웃 → 3. 프록시가 만든 504에는 [CORS](term:net.cors) 헤더가 없다
@@sources
- [MDN CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)
@@section assumptions title=내 환경 추정
@@markdown
- 추정: 앞단에 인그레스가 있다
@@section verify title=확인 방법
@@code lang=bash
curl -i https://api.example.com/health
@@section fix title=대처
@@markdown
풀 크기와 타임아웃을 조정합니다.
@@end`

var known = map[string]bool{"db.connection-pool": true, "net.cors": true}

func TestParseCollectsConceptsAndSanitizesProposals(t *testing.T) {
	a, err := Parse(validMarkup, known)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(a.ConceptIDs, ",") != "db.connection-pool,net.cors" {
		t.Fatalf("concepts = %v (the concepts: line, known only, dedup)", a.ConceptIDs)
	}
	if len(a.NewConcepts) != 1 || a.NewConcepts[0].ID != "infra.probe-design" || a.NewConcepts[0].Area != "infra-ops" ||
		len(a.NewConcepts[0].Prerequisites) != 1 || len(a.NewConcepts[0].Objectives) != 1 {
		t.Fatalf("new concepts = %+v", a.NewConcepts)
	}
	body := a.Content.Sections[0].Blocks[0].Text
	if !strings.Contains(body, "(term:infra.probe-design)") || strings.Contains(body, "term:nope.nope") || !strings.Contains(body, "없는 것") {
		t.Fatalf("links must keep known + proposed ids and drop the rest to text: %q", body)
	}
	if problems := Validate(a, lecture.ValidationContext{KnownConceptIDs: known}); len(problems) != 0 {
		t.Fatalf("valid answer flagged: %v", problems)
	}
}

func TestValidateRequiresPlanSourcesAndDiagramPlacement(t *testing.T) {
	broken := strings.Replace(validMarkup, "@@sources\n- [MDN CORS](https://developer.mozilla.org/en-US/docs/Web/HTTP/CORS)\n", "", 1)
	broken = strings.Replace(broken, "@@section assumptions title=내 환경 추정\n@@markdown\n- 추정: 앞단에 인그레스가 있다\n", "", 1)
	broken = strings.Replace(broken, "풀 크기와 타임아웃을 조정합니다.", "조정합니다.\n```mermaid\nsequenceDiagram\n  A->>B: x\n```", 1)
	a, err := Parse(broken, known)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(Validate(a, lecture.ValidationContext{KnownConceptIDs: known}), "\n")
	for _, want := range []string{"expected 5 sections", "causes must cite", "mermaid diagram inside a markdown block"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestParseRejectsIncompleteMarkup(t *testing.T) {
	if _, err := Parse(strings.TrimSuffix(validMarkup, "@@end"), known); err == nil {
		t.Fatal("markup without @@end must not parse")
	}
}

func TestRelatedConceptsFallBackToLinksAndAreCapped(t *testing.T) {
	many := map[string]bool{}
	var links strings.Builder
	for _, id := range []string{"a.one", "a.two", "a.three", "a.four", "a.five", "a.six", "a.seven"} {
		many[id] = true
		links.WriteString("[x](term:" + id + ") ")
	}
	text := strings.Replace(validMarkup, "concepts: db.connection-pool, net.cors, unknown.id\n", "", 1)
	text = strings.Replace(text, "1. 풀 고갈", links.String()+"1. 풀 고갈", 1)
	a, err := Parse(text, many)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.ConceptIDs) != MaxRelatedConcepts || a.ConceptIDs[0] != "a.one" {
		t.Fatalf("concepts = %v", a.ConceptIDs)
	}
}
