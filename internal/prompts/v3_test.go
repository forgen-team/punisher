package prompts

import (
	"strings"
	"testing"
)

func TestLabRule(t *testing.T) {
	if r := LabRule("none"); !strings.Contains(r, "쓰지 마세요") {
		t.Fatalf("none: %q", r)
	}
	if r := LabRule("linux"); !strings.Contains(r, "os=linux") || strings.Contains(r, "os=macos") {
		t.Fatalf("linux: %q", r)
	}
	if r := LabRule(""); !strings.Contains(r, "os=macos 와 os=linux") {
		t.Fatalf("library: %q", r)
	}
}

func TestLearnerSignalsOnlyWhenGiven(t *testing.T) {
	var b strings.Builder
	writeLearnerSignals(&b, nil)
	if b.Len() != 0 {
		t.Fatal("no signals must write nothing")
	}
	writeLearnerSignals(&b, []string{"원리 — 질문 2개"})
	if !strings.Contains(b.String(), "막힌 곳") || !strings.Contains(b.String(), "- 원리 — 질문 2개") {
		t.Fatalf("signals block = %q", b.String())
	}
}
