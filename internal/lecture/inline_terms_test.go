package lecture

import (
	"strings"
	"testing"
)

func TestNormalizeInlineTerms(t *testing.T) {
	c := Content{Sections: []Section{{Blocks: []Block{
		{Type: BlockMarkdown, Text: "[프로세스](term:os.process)는 실행 중인 프로그램이고 [감독자](term:os.nope)는 없다. [문서](https://x)"},
		{Type: BlockCode, Text: "[a](term:os.nope)"},
		{Type: BlockMisconception, Text: "[스레드](term:os.thread)"},
	}}}}
	dropped := NormalizeInlineTerms(&c, map[string]bool{"os.process": true, "os.thread": true})
	if dropped != 1 {
		t.Fatalf("dropped = %d", dropped)
	}
	got := c.Sections[0].Blocks[0].Text
	if got != "[프로세스](term:os.process)는 실행 중인 프로그램이고 감독자는 없다. [문서](https://x)" {
		t.Fatalf("text = %q", got)
	}
	if c.Sections[0].Blocks[1].Text != "[a](term:os.nope)" {
		t.Fatal("code blocks must be left alone")
	}
	if ids := InlineTermLinks(c); len(ids) != 2 || ids[0] != "os.process" || ids[1] != "os.thread" {
		t.Fatalf("links = %v", ids)
	}
}

func TestUnclosedFence(t *testing.T) {
	cases := map[string]bool{
		"본문\n```mermaid\nflowchart LR\n  A --> B\n```\n끝": false,
		"본문\n```mermaid\nflowchart LR\n### 2. 집계법\n본문 계속": true,
		"~~~\ncode\n~~~":           false,
		"````\n```\n````":          false,
		"인라인 `code` 는 상관없음":        false,
		"```go\nx := 1\n``` 뒤에 글자": true,
	}
	for text, want := range cases {
		if got := UnclosedFence(text); got != want {
			t.Errorf("UnclosedFence(%q) = %v, want %v", text, got, want)
		}
	}
	c := Content{Sections: []Section{{Kind: "principles", Blocks: []Block{{Type: BlockMarkdown, Text: "a\n```mermaid\nB"}}}}}
	found := false
	for _, i := range Validate(c, ValidationContext{}) {
		found = found || strings.Contains(i.Message, "unclosed")
	}
	if !found {
		t.Fatal("validate must flag an unclosed fence")
	}
}

func TestCodeLangWithoutKey(t *testing.T) {
	markup := "@@lecture\ntitle: t\nobjectives: c#1\nnew_scope: x\n@@section situation_question minutes=30 title=a\n@@code bash\nls\n@@log lang=text\nout\n@@end"
	c, err := ParseMarkup(markup, "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.Sections[0].Blocks[0].Lang; got != "bash" {
		t.Fatalf("lang = %q", got)
	}
}

func TestSourceLineWithTrailingNote(t *testing.T) {
	markup := "@@lecture\ntitle: t\nobjectives: c#1\nnew_scope: x\n@@section principles minutes=30 title=a\n@@markdown\nx\n@@sources\n- [proc(5)](https://man7.org/linux/man-pages/man5/proc.5.html) — fd·environ 의 근거\n@@end"
	c, err := ParseMarkup(markup, "c", 1)
	if err != nil {
		t.Fatal(err)
	}
	src := c.Sections[0].Blocks[1].Sources
	if len(src) != 1 || src[0].URL != "https://man7.org/linux/man-pages/man5/proc.5.html" {
		t.Fatalf("sources = %+v", src)
	}
}
