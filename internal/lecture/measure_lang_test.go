package lecture

import "testing"

func TestMeasureLanguage(t *testing.T) {
	ko := Content{Sections: []Section{{Blocks: []Block{{Type: BlockMarkdown, Text: "프로세스는 실행 중인 프로그램이다. process 라는 말도 쓴다."}}}}}
	en := Content{Sections: []Section{{Blocks: []Block{{Type: BlockMarkdown, Text: "A process is a running program (프로세스)."}}}}}
	if got := MeasureContent(ko).Language(); got != "ko" {
		t.Fatalf("ko lecture: %s", got)
	}
	if got := MeasureContent(en).Language(); got != "en" {
		t.Fatalf("en lecture: %s", got)
	}
	if got := MeasureContent(Content{}).Language(); got != "ko" {
		t.Fatalf("empty: %s", got)
	}
	if lo, hi := ProseBounds("en"); lo <= MinProseChars || hi <= MaxProseChars {
		t.Fatal("English bounds should be larger")
	}
}

func TestMermaidSwallowedEnglishProse(t *testing.T) {
	src := "flowchart LR\n  A --> B\n\nThe client sends a request to the server first.\nThen the server answers with the cached value here.\n"
	if MermaidSwallowedProse(src) == "" {
		t.Fatal("English prose after a diagram should be caught")
	}
	if got := MermaidSwallowedProse("flowchart LR\n  A[Client] --> B[Server]\n\n  B --> C[Cache store]\n"); got != "" {
		t.Fatalf("diagram flagged: %s", got)
	}
}
