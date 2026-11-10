package lecture

import (
	"strings"
	"testing"
)

// 2026-11-10: 운영 강의 본문의 "이 점은 @@term 에서 다루는 ALU 설계"가 화면에서 빈칸이 됐다.
func TestStrayDirective(t *testing.T) {
	cases := map[string]string{
		"이 점은 @@term 에서 다루는 ALU 설계와도 이어집니다.":    "@@term",
		"새 @@term{프로세스}를 하나 만듭니다.":              "@@term",
		"- → @@term 부동소수점":                      "@@term",
		"  @@code lang=go 들여쓴 지시어":              "@@code",
		"[프로세스](term:os.process)는 그대로":          "",
		"인라인 `@@term` 설명은 괜찮다":                  "",
		"```text\n@@term concept=a term=b\n```": "",
		"diff 헤더 @@ -1,2 +1,2 @@ 는 지시어가 아니다":    "",
		"SELECT @@version 은 지시어가 아니다":           "",
	}
	for text, want := range cases {
		if got := StrayDirective(text); got != want {
			t.Errorf("StrayDirective(%q) = %q, want %q", text, got, want)
		}
	}
}

func TestValidateBlockRejectsStrayDirective(t *testing.T) {
	for _, typ := range []string{BlockMarkdown, BlockMisconception} {
		issues := ValidateBlock("b", Block{Type: typ, Text: "이 점은 @@term 에서 다룹니다."}, ValidationContext{})
		if len(issues) != 1 || !strings.Contains(issues[0].Message, "@@term") {
			t.Errorf("%s: issues = %v", typ, issues)
		}
	}
	if issues := ValidateBlock("b", Block{Type: BlockCode, Lang: "text", Text: "@@term 에서"}, ValidationContext{}); len(issues) != 0 {
		t.Errorf("code block: issues = %v", issues)
	}
}
