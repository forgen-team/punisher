package prompts

import (
	"fmt"
	"strings"
	"testing"
	"unicode/utf8"
)

const gapHeading = "## 학습자의 지식 카드에서 모르는 것·충돌하는 것"

// The gap section is written like the misconception one: present with lines,
// absent when the list is empty, in both lecture and outline prompts.
func TestKnowledgeGapsSection(t *testing.T) {
	c := LectureContext{ConceptID: "c", ConceptName: "인덱스", Depth: 1,
		KnowledgeGaps: []string{"모름: 복합 인덱스 순서가 왜 중요한가", "충돌: 인덱스는 항상 빠르다 — ① 읽기는 빨라진다 ② 쓰기는 느려진다"}}
	for name, text := range map[string]string{"lecture": Lecture(c, "", "", Defaults()), "outline": Outline(c, Defaults())} {
		if !strings.Contains(text, gapHeading) || !strings.Contains(text, "충돌이면 어떤 조건에서 각각 맞는지") {
			t.Fatalf("%s: section missing:\n%s", name, text)
		}
		for _, g := range c.KnowledgeGaps {
			if !strings.Contains(text, "- "+g+"\n") {
				t.Fatalf("%s: line %q missing", name, g)
			}
		}
	}
	c.KnowledgeGaps = nil
	if strings.Contains(Lecture(c, "", "", Defaults()), gapHeading) {
		t.Fatal("no gaps → no section")
	}
	c.KnowledgeGaps = []string{"  ", ""}
	if strings.Contains(Lecture(c, "", "", Defaults()), gapHeading) {
		t.Fatal("blank gaps → no section")
	}
}

func TestKnowledgeGapsCapped(t *testing.T) {
	var gaps []string
	for i := 0; i < MaxKnowledgeGaps+4; i++ {
		gaps = append(gaps, fmt.Sprintf("모름: 질문%02d %s", i, strings.Repeat("가", 400)))
	}
	var b strings.Builder
	writeKnowledgeGaps(&b, gaps)
	lines := strings.Split(strings.TrimSpace(b.String()), "\n")[1:]
	if len(lines) != MaxKnowledgeGaps {
		t.Fatalf("lines = %d, want %d", len(lines), MaxKnowledgeGaps)
	}
	for _, l := range lines {
		if n := utf8.RuneCountInString(strings.TrimPrefix(l, "- ")); n > MaxKnowledgeGapRunes+1 {
			t.Fatalf("line of %d runes exceeds the cap", n)
		}
	}
	if strings.Contains(b.String(), fmt.Sprintf("질문%02d", MaxKnowledgeGaps)) {
		t.Fatal("gaps after the cap must be dropped")
	}
}
