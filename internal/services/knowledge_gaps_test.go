package services

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
)

const gapContext = `{"kind":"knowledge_gap","gaps":[
	{"type":"unknown","text":"복합 인덱스에서  컬럼 순서가 왜 중요한가","grade":"unknown"},
	{"type":"conflict","text":"인덱스는 항상 조회를 빠르게 한다","sides":["읽기 위주 표에서는 빨라진다","쓰기 많은 표에서는 오히려 느려진다"],"grade":"case"},
	{"type":"conflict","text":"한쪽만 있는 충돌","sides":["A"]},
	{"type":"other","text":"모르는 타입은 버린다"},
	{"type":"unknown","text":"   "}
]}`

func TestKnowledgeGapsFromContext(t *testing.T) {
	got := KnowledgeGapsFromContext(json.RawMessage(gapContext))
	want := []string{
		"모름: 복합 인덱스에서 컬럼 순서가 왜 중요한가",
		"충돌: 인덱스는 항상 조회를 빠르게 한다 — ① 읽기 위주 표에서는 빨라진다 ② 쓰기 많은 표에서는 오히려 느려진다 (근거 등급: 사례)",
		"충돌: 한쪽만 있는 충돌",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("gaps =\n%s", strings.Join(got, "\n"))
	}
	for name, raw := range map[string]string{
		"other kind": `{"kind":"rss_item","gaps":[{"type":"unknown","text":"x"}]}`,
		"no kind":    `{"gaps":[{"type":"unknown","text":"x"}]}`,
		"broken":     `{"kind":"knowledge_gap","gaps":[`,
		"empty":      `{}`,
		"nil":        ``,
		"not object": `"knowledge_gap"`,
	} {
		if g := KnowledgeGapsFromContext(json.RawMessage(raw)); g != nil {
			t.Errorf("%s: gaps = %v, want none", name, g)
		}
	}
}

func TestKnowledgeGapsFromContextCaps(t *testing.T) {
	var gaps []string
	for i := 0; i < prompts.MaxKnowledgeGaps+3; i++ {
		gaps = append(gaps, `{"type":"conflict","text":"`+strings.Repeat("가", 500)+`","sides":["`+strings.Repeat("나", 200)+`","`+strings.Repeat("다", 200)+`"]}`)
	}
	got := KnowledgeGapsFromContext(json.RawMessage(`{"kind":"knowledge_gap","gaps":[` + strings.Join(gaps, ",") + `]}`))
	if len(got) != prompts.MaxKnowledgeGaps {
		t.Fatalf("gaps = %d", len(got))
	}
	if n := len([]rune(got[0])); n > prompts.MaxKnowledgeGapRunes || !strings.Contains(got[0], "② ") {
		t.Fatalf("a capped conflict line keeps both sides within %d runes: %d %q", prompts.MaxKnowledgeGapRunes, n, got[0])
	}
}

// Gaps reach user-owned generations only; library lectures never carry them.
func TestKnowledgeGapsPersonalGate(t *testing.T) {
	gaps := KnowledgeGapsFromContext(json.RawMessage(gapContext))
	lib := personalTarget("")
	lib.personal.gaps = gaps
	if pc := lib.LectureContext(); len(pc.KnowledgeGaps) != 0 || strings.Contains(prompts.Lecture(pc, "", "", prompts.Defaults()), "지식 카드에서 모르는 것") {
		t.Fatal("library prompt must not carry knowledge gaps")
	}
	own := personalTarget("u1")
	own.personal.gaps = gaps
	text := prompts.Lecture(own.LectureContext(), "", "", prompts.Defaults())
	if !strings.Contains(text, "## 학습자의 지식 카드에서 모르는 것·충돌하는 것") || !strings.Contains(text, "- "+gaps[1]+"\n") {
		t.Fatalf("user-owned prompt lacks the gap section:\n%s", text)
	}
	if text := prompts.Lecture(personalTarget("u1").LectureContext(), "", "", prompts.Defaults()); strings.Contains(text, "지식 카드에서 모르는 것") {
		t.Fatal("no gaps → no section")
	}
}

// studyRequestGaps reads the user's own stored request: knowledge_gap context
// gives gaps, another kind or another user's request gives none.
func TestStudyRequestGapsFromStore(t *testing.T) {
	s, _, user, _ := checkService(t)
	ctx := context.Background()
	save := func(owner, key, raw string) string {
		t.Helper()
		r, _, err := s.Store.CreateStudyRequest(ctx, models.StudyRequest{UserID: owner, Source: "rss", SourceRef: "claim:1",
			Query: "인덱스는 항상 빠른가", Context: json.RawMessage(raw), IdempotencyKey: key, Status: "unresolved"})
		if err != nil {
			t.Fatal(err)
		}
		return r.ID
	}
	gapID := save(user, "claim:1:conflict", gapContext)
	if got := s.studyRequestGaps(ctx, user, gapID); len(got) != 3 || !strings.HasPrefix(got[1], "충돌: ") {
		t.Fatalf("gaps = %v", got)
	}
	otherKind := save(user, "rss:item", `{"kind":"rss_item","url":"https://example.com"}`)
	if got := s.studyRequestGaps(ctx, user, otherKind); got != nil {
		t.Fatalf("other kind gaps = %v", got)
	}
	if got := s.studyRequestGaps(ctx, "someone-else", gapID); got != nil {
		t.Fatalf("another user's request must not leak: %v", got)
	}
	if got := s.studyRequestGaps(ctx, user, ""); got != nil {
		t.Fatal("no request id → no gaps")
	}

	// The prompt preview shows the section (example lines when the user has no request).
	pv, err := s.PreviewPrompt(ctx, user, prompts.BlockLectureFormat, nil)
	if err != nil || !strings.Contains(pv.User, "## 학습자의 지식 카드에서 모르는 것·충돌하는 것") {
		t.Fatalf("prompt preview shows the gap block (%v)", err)
	}
}
