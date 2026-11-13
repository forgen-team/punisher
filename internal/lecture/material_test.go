package lecture

import (
	"encoding/json"
	"strings"
	"testing"
)

// filler is prose that brings a test lecture into the §4 volume bounds.
func filler(chars int) string {
	const sentence = "정수는 고정된 비트 폭 안에서만 값을 담습니다. "
	per := visibleChars(sentence)
	return strings.Repeat(sentence, chars/per+1)
}

// materialLecture is a new-format lecture that passes Validate and MachineCheck.
func materialLecture() string {
	return `@@lecture
title: 32비트 정수가 넘치면 무슨 일이 생기나
objectives: os.process#1, os.process#2
new_scope: 고정 폭 정수의 범위와 넘침
prerequisites: os.kernel-and-syscall
@@section situation_question minutes=2 title=주문 번호가 음수가 됐다
@@markdown
` + filler(1500) + `
@@log lang=text
ERROR: integer out of range
@@section principles minutes=9 title=고정 폭 정수
@@keypoint kind=definition
**부호 없는 이진수**는 각 자리의 2의 거듭제곱을 더한 값입니다. 예: 101₂ = 5. 비예: 부호 비트가 있는 표현.
@@trace
{"caption":"4비트 덧셈","columns":["단계","값"],"rows":[["1","0111"],["2","1000"]],"note":"자리올림이 부호 비트로 넘어간다"}
@@exercise
0110 + 0011 의 4비트 결과는?
@@answer
1001 입니다.
@@markdown
` + filler(2500) + `
@@chart
{"kind":"line","title":"비트 수와 최댓값","x":{"label":"비트","values":[8,16,32]},"y":{"label":"최댓값","scale":"log"},"series":[{"name":"부호 없음","values":[255,65535,4294967295]}]}
@@sources
- [Integer types](https://www.postgresql.org/docs/current/datatype-numeric.html)
@@section walkthrough minutes=6 title=장면 재현
@@code lang=sql
SELECT 2147483647::int + 1;
@@pause
위 쿼리는 무엇을 출력할까요?
@@log lang=text
ERROR:  integer out of range
@@diff lang=sql
{"before":"id integer","after":"id bigint","note":"시퀀스 상한도 함께 바꾼다"}
@@exercise
아래 풀이에서 틀린 곳을 찾으세요.
@@answer
자리올림과 넘침을 혼동했습니다.
@@markdown
` + filler(2000) + `
@@sources
- [Sequences](https://www.postgresql.org/docs/current/sql-createsequence.html)
@@section judgment_next minutes=3 title=판단과 다음
@@markdown
` + filler(1000) + `
@@summary
- 고정 폭 정수는 범위가 있다
- 넘침은 자리올림과 다르다
- 시퀀스 상한을 함께 본다
@@end`
}

func TestParseMaterialBlocks(t *testing.T) {
	c := mustParse(t, materialLecture())
	p := c.Sections[1].Blocks
	if p[0].Type != BlockKeypoint || p[0].Kind != KeypointDefinition || !strings.Contains(p[0].Text, "부호 없는 이진수") {
		t.Fatalf("keypoint: %+v", p[0])
	}
	var tr TraceData
	if p[1].Type != BlockTrace || p[1].Text != "" || json.Unmarshal(p[1].Data, &tr) != nil || len(tr.Rows) != 2 || tr.Columns[0] != "단계" {
		t.Fatalf("trace: %+v", p[1])
	}
	w := c.Sections[2].Blocks
	if w[1].Type != BlockPause || w[1].Text != "위 쿼리는 무엇을 출력할까요?" {
		t.Fatalf("pause: %+v", w[1])
	}
	if w[3].Type != BlockDiff || w[3].Lang != "sql" || !strings.Contains(string(w[3].Data), `"after":"id bigint"`) {
		t.Fatalf("diff: %+v", w[3])
	}
	if issues := Validate(c, ctxFor()); len(issues) != 0 {
		t.Fatalf("valid material lecture has issues: %v", issues)
	}
	if issues := MachineCheck(c); len(issues) != 0 {
		t.Fatalf("valid material lecture fails machine checks: %v", issues)
	}
}

func TestStructuredJSONFenceTolerated(t *testing.T) {
	src := strings.Replace(materialLecture(), `@@trace
{"caption"`, "@@trace\n```json\n{\"caption\"", 1)
	src = strings.Replace(src, `"자리올림이 부호 비트로 넘어간다"}`, "\"자리올림이 부호 비트로 넘어간다\"}\n```", 1)
	c := mustParse(t, src)
	if len(c.Sections[1].Blocks[1].Data) == 0 {
		t.Fatalf("fenced trace JSON must parse: %+v", c.Sections[1].Blocks[1])
	}
}

func TestMaterialBlockValidation(t *testing.T) {
	cases := []struct {
		name, from, to, want string
	}{
		{"keypoint kind", "kind=definition", "kind=note", "keypoint kind"},
		{"trace not json", `{"caption":"4비트 덧셈"`, `캡션 {"caption":"4비트 덧셈"`, "trace block body must be one JSON object"},
		{"trace row length", `["2","1000"]`, `["2"]`, "row 2 has 1 cells"},
		{"trace too few rows", `,["2","1000"]`, ``, "trace needs 2..16 rows"},
		{"trace numbers", `["1","0111"]`, `[1,"0111"]`, "trace JSON does not match"},
		{"trace unknown key", `"note":"자리올림`, `"notes":"자리올림`, "trace JSON does not match"},
		{"chart length", `"values":[255,65535,4294967295]`, `"values":[255,65535]`, "series 1 has 2 values"},
		{"chart log non-positive", `"values":[255,65535,4294967295]`, `"values":[0,65535,4294967295]`, "<= 0 on a log scale"},
		{"chart kind", `"kind":"line"`, `"kind":"pie"`, "chart kind"},
		{"chart scale", `"scale":"log"`, `"scale":"exp"`, "y.scale"},
		{"diff same", `"after":"id bigint"`, `"after":"id integer"`, "before and after are the same"},
		{"diff lang", "@@diff lang=sql", "@@diff", "diff block needs lang"},
		{"pause not before output", "@@pause\n위 쿼리는 무엇을 출력할까요?\n@@log lang=text", "@@pause\n위 쿼리는 무엇을 출력할까요?\n@@markdown\n설명\n@@log lang=text", "@@pause must be followed"},
		{"exercise without answer", "@@exercise\n0110 + 0011 의 4비트 결과는?\n@@answer\n1001 입니다.\n", "@@exercise\n0110 + 0011 의 4비트 결과는?\n", "@@exercise must be followed"},
		{"answer alone", "@@exercise\n0110 + 0011 의 4비트 결과는?\n", "", "@@answer must come right after"},
		{"summary bullets", "- 시퀀스 상한을 함께 본다\n", "", "summary needs 3..6 bullets"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := materialLecture()
			if !strings.Contains(src, tc.from) {
				t.Fatalf("fixture lacks %q", tc.from)
			}
			c := mustParse(t, strings.Replace(src, tc.from, tc.to, 1))
			if issues := Validate(c, ctxFor()); !hasIssue(issues, tc.want) {
				t.Fatalf("want issue %q, got %v", tc.want, issues)
			}
		})
	}
}

// TestLegacyLectureStillValid: a lecture without material blocks parses and
// validates (stored lectures keep rendering; composition is a machine check only).
func TestLegacyLectureStillValid(t *testing.T) {
	c := mustParse(t, sample)
	if HasMaterialBlocks(c) {
		t.Fatal("sample has no material blocks")
	}
	if issues := Validate(c, ctxFor()); len(issues) != 0 {
		t.Fatalf("legacy lecture: %v", issues)
	}
	raw, err := json.Marshal(c)
	if err != nil || strings.Contains(string(raw), `"data"`) || strings.Contains(string(raw), `"kind":"definition"`) {
		t.Fatalf("legacy JSON must not grow new keys: %s %v", raw, err)
	}
}

func TestRenderMarkupRoundTrip(t *testing.T) {
	for _, src := range []string{sample, materialLecture()} {
		a := mustParse(t, src)
		b := mustParse(t, RenderMarkup(a))
		ha, _ := Hash(a)
		hb, _ := Hash(b)
		if ha != hb {
			t.Fatalf("round trip changed the lecture:\n%s", RenderMarkup(a))
		}
	}
}
