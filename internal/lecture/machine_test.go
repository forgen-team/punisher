package lecture

import (
	"strings"
	"testing"
)

func TestMachineCheck(t *testing.T) {
	cases := []struct {
		name, from, to, want string
	}{
		{"self link", "**부호 없는 이진수**는", "[프로세스](term:os.process)는", "own concept (term:os.process)"},
		{"meta phrase", "자리올림과 넘침을 혼동했습니다.", "MDN이 아닌 설명으로 보면 자리올림과 넘침을 혼동했습니다.", `meta phrase "MDN이 아닌"`},
		{"other lecture", "자리올림과 넘침을 혼동했습니다.", "이 부분은 다른 강의에서 다룹니다.", "다른 강의에서 다룹"},
		{"section id", "시퀀스 상한을 함께 본다", "walkthrough 에서 본 시퀀스 상한", `section id "walkthrough"`},
		{"no definition", "@@keypoint kind=definition", "@@keypoint kind=rule", "@@keypoint kind=definition"},
		{"no trace", "@@trace\n{\"caption\":\"4비트 덧셈\",\"columns\":[\"단계\",\"값\"],\"rows\":[[\"1\",\"0111\"],[\"2\",\"1000\"]],\"note\":\"자리올림이 부호 비트로 넘어간다\"}\n", "", "at least one @@trace"},
		{"one exercise", "@@exercise\n아래 풀이에서 틀린 곳을 찾으세요.\n@@answer\n자리올림과 넘침을 혼동했습니다.\n", "", "at least 2 @@exercise"},
		{"no pause", "@@pause\n위 쿼리는 무엇을 출력할까요?\n", "", "at least 1 @@pause"},
		{"no summary", "@@summary\n- 고정 폭 정수는 범위가 있다\n- 넘침은 자리올림과 다르다\n- 시퀀스 상한을 함께 본다\n", "", "exactly one @@summary"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := materialLecture()
			if !strings.Contains(src, tc.from) {
				t.Fatalf("fixture lacks %q", tc.from)
			}
			if issues := MachineCheck(mustParse(t, strings.Replace(src, tc.from, tc.to, 1))); !hasIssue(issues, tc.want) {
				t.Fatalf("want issue %q, got %v", tc.want, issues)
			}
		})
	}
}

func TestMachineCheckExemptsCode(t *testing.T) {
	src := strings.Replace(materialLecture(), "SELECT 2147483647::int + 1;", "-- principles walkthrough MDN이 아닌\nSELECT 2147483647::int + 1;", 1)
	src = strings.Replace(src, "자리올림과 넘침을 혼동했습니다.", "`judgment_next` 는 코드 표기입니다.", 1)
	if issues := MachineCheck(mustParse(t, src)); len(issues) != 0 {
		t.Fatalf("code and inline code are not reader prose: %v", issues)
	}
}

func TestMachineCheckSummaryOutsideJudgment(t *testing.T) {
	src := strings.Replace(materialLecture(), "@@sources\n- [Sequences]", "@@summary\n- a\n- b\n- c\n@@sources\n- [Sequences]", 1)
	if issues := MachineCheck(mustParse(t, src)); !hasIssue(issues, "belongs in judgment_next") || !hasIssue(issues, "exactly one @@summary") {
		t.Fatalf("summary outside judgment_next: %v", issues)
	}
}

func TestMachineCheckVolume(t *testing.T) {
	short := strings.Replace(materialLecture(), filler(2500), "짧다", 1)
	if issues := MachineCheck(mustParse(t, short)); !hasIssue(issues, "prose must be 6500..11000") {
		t.Fatalf("short lecture: %v", issues)
	}
	long := strings.Replace(materialLecture(), filler(2500), filler(9000), 1)
	if issues := MachineCheck(mustParse(t, long)); !hasIssue(issues, "prose must be 6500..11000") {
		t.Fatalf("long lecture: %v", issues)
	}
}

func TestMeasureContent(t *testing.T) {
	c := Content{Sections: []Section{{Blocks: []Block{
		{Type: BlockMarkdown, Text: "가나 다\n```go\nx := 1\n```\n라"},
		{Type: BlockCode, Text: "ab c"},
		{Type: BlockExercise, Text: "문제"},
		{Type: BlockPause, Text: "질문"},
		{Type: BlockTrace, Data: []byte(`{"caption":"표","columns":["a","b"],"rows":[["1","2"],["3","4"],["5","6"]]}`)},
		{Type: BlockDiff, Data: []byte(`{"before":"a b","after":"cd"}`)},
		{Type: BlockMermaid, Text: "graph TD; A-->B"},
	}}}}
	m := MeasureContent(c)
	// prose: 가나다라(4) + 문제(2) + 질문(2) + 표(1) = 9; code: x:=1(4) + abc(3) + ab(2) + cd(2) = 11
	if m.ProseChars != 9 || m.CodeChars != 11 || m.Exercises != 1 || m.Pauses != 1 || m.TraceRows != 3 {
		t.Fatalf("measure = %+v", m)
	}
}
