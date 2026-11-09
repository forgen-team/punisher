package lecture

import (
	"strings"
	"testing"
)

// Fixture modeled on lecture 46fdbcc8…: the diagram's end was missed and section 3's
// markdown (heading, prose, a raw @@term mention) fell into the mermaid source.
const swallowedMermaid = "flowchart LR\n" +
	"  A[클라이언트] -->|요청| B[로드밸런서]\n" +
	"  B --> C[서버]\n" +
	"\n" +
	"## 3. 코드로 확인하기\n" +
	"이제 실제로 요청이 어떻게 흘러가는지 코드로 봅니다.\n" +
	"앞에서 본 @@term concept=net.tcp term=TCP 연결이 여기서 다시 등장합니다.\n"

func TestMermaidSwallowedProse(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string // substring of the reason, "" = clean
	}{
		{"clean flowchart with Korean labels and a blank line", "flowchart LR\n  A[요청 시작] --> B{캐시 있음?}\n\n  B -- 예 --> C[바로 응답]\n  B -- 아니오 --> D[DB 조회]\n  subgraph 서버\n  D\n  end", ""},
		{"clean sequence diagram", "sequenceDiagram\n  participant 브라우저\n  브라우저->>서버: GET /\n\n  Note over 서버: 처리", ""},
		{"swallowed section (fixture)", swallowedMermaid, "markdown heading"},
		{"directive inside", "graph TD\n  A --> B\n  설명은 @@term concept=os.process term=프로세스 참고", "@@ directive"},
		{"prose after blank line", "graph TD\n  A --> B\n\n이 그림에서 요청은 왼쪽에서 오른쪽으로 갑니다.\n각 단계는 실패할 수 있습니다.", "prose paragraph"},
		{"single Korean note line is allowed", "graph TD\n  A --> B\n\n그림 설명", ""},
	}
	for _, tc := range cases {
		got := MermaidSwallowedProse(tc.src)
		if (tc.want == "") != (got == "") || (tc.want != "" && !strings.Contains(got, tc.want)) {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestValidateFlagsSwallowedMermaid(t *testing.T) {
	c := Content{Sections: []Section{{Kind: "principles", Blocks: []Block{{Type: BlockMermaid, Text: swallowedMermaid}}}}}
	found := false
	for _, i := range Validate(c, ValidationContext{}) {
		found = found || strings.Contains(i.Message, "mermaid block swallowed lecture prose")
	}
	if !found {
		t.Fatal("validate must flag a mermaid block that swallowed prose")
	}
}
