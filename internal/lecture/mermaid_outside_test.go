package lecture

import (
	"strings"
	"testing"
)

// 2026-11-09: 즉석 강의에서 sequenceDiagram 이 @@mermaid 없이 본문 코드로 보였다.
func TestMermaidOutsideBlock(t *testing.T) {
	cases := map[string]bool{
		"본문\n```mermaid\nsequenceDiagram\n  A->>B: hi\n```\n끝":        true,
		"본문\n```\nsequenceDiagram\n  participant A as 서버 A\n```":      true,
		"본문\nsequenceDiagram\n  participant A as 서버 A\n  A->>S: 락 획득": true,
		"  flowchart LR\n  A --> B":                                   true,
		"graph TD\nA-->B":                                             true,
		"본문\n```bash\ngraph TD\n```":                                  false, // 다른 언어 코드는 그대로
		"sequenceDiagram 은 시간 순서를 그리는 도식입니다.":                         false, // 문장 안 언급
		"flowchart 라는 말":                                              false,
		"pie 차트로 보면":                                                  false,
		"```go\nfunc main() {}\n```":                                  false,
		"    sequenceDiagram (들여쓴 코드)":                                false,
	}
	for text, want := range cases {
		if got := MermaidOutsideBlock(text) != ""; got != want {
			t.Errorf("MermaidOutsideBlock(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestValidateFlagsMermaidInMarkdown(t *testing.T) {
	c := Content{Sections: []Section{{Kind: "principles", Blocks: []Block{{Type: BlockMarkdown, Text: "설명\n```mermaid\nsequenceDiagram\n  A->>B: x\n```"}}}}}
	found := false
	for _, i := range Validate(c, ValidationContext{}) {
		found = found || strings.Contains(i.Message, "mermaid diagram inside a markdown block")
	}
	if !found {
		t.Fatal("validate must flag a mermaid diagram written inside a markdown block")
	}
}
