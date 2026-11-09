package lecture

import (
	"errors"
	"strings"
	"testing"
)

const sample = `서두 잡담은 무시된다
@@lecture
title: 프로세스는 무엇을 가지고 있나
objectives: os.process#1, os.process#2
new_scope: 프로세스가 가진 자원과 PID
prerequisites: os.kernel-and-syscall
@@section situation_question minutes=3 title=상황과 오늘의 질문
배포 후 서비스가 사라졌다.
@@section principles minutes=12 title=배경과 원리
@@markdown
프로세스는 실행 중인 프로그램이다.
@@term concept=os.kernel-and-syscall term=시스템 콜
커널에 일을 요청하는 경계
@@term concept=os.ebpf proposed term=eBPF
커널 안에서 안전하게 실행되는 프로그램
@@misconception
스레드와 프로세스는 같다? 아니다.
@@sources
- [fork(2)](https://man7.org/linux/man-pages/man2/fork.2.html)
@@section walkthrough minutes=10 title=코드와 로그로 보기
@@code lang=bash
ps -ef --forest
@@ -1,2 +1,2 @@ diff hunk stays in the block
@@mermaid
graph TD; A-->B
@@sources
- [ps(1)](https://man7.org/linux/man-pages/man1/ps.1.html)
@@section judgment_next minutes=5 title=판단 기준과 다음 연결
@@markdown
다음은 컨테이너.
@@end
이후 텍스트는 무시`

func ctxFor() ValidationContext {
	return ValidationContext{
		KnownConceptIDs:     map[string]bool{"os.process": true, "os.kernel-and-syscall": true},
		PlannedObjectiveIDs: []string{"os.process#1", "os.process#2"},
		AllowedObjectiveIDs: map[string]bool{"os.process#1": true, "os.process#2": true, "os.process#3": true},
	}
}

func TestParseMarkupBuildsSchema(t *testing.T) {
	c, err := ParseMarkup(sample, "os.process", 1)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.Title == "" || len(c.Sections) != 4 || c.EstimatedMinutes != 30 || c.ConceptID != "os.process" {
		t.Fatalf("unexpected content: %+v", c)
	}
	principles := c.Sections[1]
	if principles.Blocks[1].Type != BlockTerm || principles.Blocks[1].Term != "시스템 콜" || principles.Blocks[1].Proposed {
		t.Fatalf("term block: %+v", principles.Blocks[1])
	}
	if !principles.Blocks[2].Proposed || principles.Blocks[2].ConceptID != "os.ebpf" {
		t.Fatalf("proposed term block: %+v", principles.Blocks[2])
	}
	code := c.Sections[2].Blocks[0]
	if code.Lang != "bash" || !strings.Contains(code.Text, "@@ -1,2") {
		t.Fatalf("code block: %+v", code)
	}
	if c.Sections[0].Blocks[0].Type != BlockMarkdown || c.Sections[0].Blocks[0].Anchor != "situation_question-1" {
		t.Fatalf("loose text should become markdown: %+v", c.Sections[0].Blocks)
	}
	if got := len(c.AllSources()); got != 2 {
		t.Fatalf("sources: %d", got)
	}
	if issues := Validate(c, ctxFor()); len(issues) != 0 {
		t.Fatalf("valid sample has issues: %v", issues)
	}
	if len(ProposedTerms(c)) != 1 {
		t.Fatalf("proposed terms")
	}
}

func TestParseMarkupIncomplete(t *testing.T) {
	truncated := sample[:strings.Index(sample, "@@end")]
	if _, err := ParseMarkup(truncated, "os.process", 1); !errors.Is(err, ErrIncomplete) {
		t.Fatalf("want ErrIncomplete, got %v", err)
	}
}

func TestParseMarkupUnknownDirective(t *testing.T) {
	if _, err := ParseMarkup("@@section principles minutes=3 title=x\n@@video\n@@end", "c", 1); err == nil {
		t.Fatal("unknown directive must fail")
	}
}

func mustParse(t *testing.T, text string) Content {
	t.Helper()
	c, err := ParseMarkup(text, "os.process", 1)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	return c
}

func hasIssue(issues []Issue, substr string) bool {
	for _, i := range issues {
		if strings.Contains(i.String(), substr) {
			return true
		}
	}
	return false
}

func TestValidateRules(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(string) string
		want   string
	}{
		{"missing section", func(s string) string {
			start := strings.Index(s, "@@section walkthrough")
			end := strings.Index(s, "@@section judgment_next")
			return s[:start] + s[end:]
		}, "expected 4 sections"},
		{"wrong order", func(s string) string {
			return strings.Replace(s, "@@section situation_question", "@@section principles_x", 1)
		}, `expected section "situation_question"`},
		{"too short", func(s string) string { return strings.Replace(s, "minutes=12", "minutes=2", 1) }, "total minutes"},
		{"too long", func(s string) string { return strings.Replace(s, "minutes=12", "minutes=20", 1) }, "total minutes"},
		{"unknown term", func(s string) string {
			return strings.Replace(s, "concept=os.kernel-and-syscall term", "concept=os.nope term", 1)
		}, "does not exist and is not marked proposed"},
		{"no sources in factual", func(s string) string {
			return strings.Replace(s, "@@sources\n- [ps(1)](https://man7.org/linux/man-pages/man1/ps.1.html)\n", "", 1)
		}, `factual section "walkthrough" must cite`},
		{"http source", func(s string) string {
			return strings.Replace(s, "https://man7.org/linux/man-pages/man1", "http://man7.org/x", 1)
		}, "must be https"},
		{"code without lang", func(s string) string { return strings.Replace(s, "@@code lang=bash", "@@code", 1) }, "needs lang"},
		{"uncovered objective", func(s string) string {
			return strings.Replace(s, "objectives: os.process#1, os.process#2", "objectives: os.process#1", 1)
		}, `"os.process#2" is not covered`},
		{"foreign objective", func(s string) string { return strings.Replace(s, "os.process#2\n", "os.process#2, db.index#1\n", 1) }, "does not belong"},
		{"no new scope", func(s string) string {
			return strings.Replace(s, "new_scope: 프로세스가 가진 자원과 PID\n", "", 1)
		}, "newScope"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			issues := Validate(mustParse(t, tc.mutate(sample)), ctxFor())
			if !hasIssue(issues, tc.want) {
				t.Fatalf("want issue %q, got %v", tc.want, issues)
			}
		})
	}
}

func TestHashStableAndSensitive(t *testing.T) {
	a := mustParse(t, sample)
	b := mustParse(t, sample)
	ha, _ := Hash(a)
	hb, _ := Hash(b)
	if ha != hb || len(ha) != 64 {
		t.Fatalf("hash not stable: %s %s", ha, hb)
	}
	b.Title += "!"
	if hc, _ := Hash(b); hc == ha {
		t.Fatal("hash must change with content")
	}
}
