package services

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/chapter"
	"github.com/forgen-team/punisher/internal/llm"
)

type failingSignals struct{ calls int }

func (f *failingSignals) LearningSignals(context.Context, string) (LearningSignals, error) {
	f.calls++
	return LearningSignals{}, errors.New("rss down")
}

type sequenceLLM struct {
	texts []string
	users []string
}

func (s *sequenceLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	s.users = append(s.users, req.User)
	text := s.texts[0]
	if len(s.texts) > 1 {
		s.texts = s.texts[1:]
	}
	return llm.MessagesResult{Text: text, Model: "fake"}, nil
}

func (s *sequenceLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return s.Create(ctx, key, req)
}

func TestSuggestChaptersWithoutRSS(t *testing.T) {
	s, user := dbService(t)
	sig := &failingSignals{}
	s.Signals = sig
	list, err := s.SuggestChapters(context.Background(), user)
	if err != nil {
		t.Fatalf("an RSS failure must not fail suggestions: %v", err)
	}
	if sig.calls != 1 || len(list) < 3 {
		t.Fatalf("calls=%d suggestions=%+v", sig.calls, list)
	}
}

func TestCreateChapterDraftRepairsOnce(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	seed := firstConceptWithObjectives(t, s) // a shared seed concept (fresh SQLite DB)
	concept := func(key string, prereq string) string {
		return `{"key":"` + key + `","name":"` + key + ` 개념","area":"os","depth":1,"summary":"s","objectives":[{"text":"설명한다","depth":1}],"prerequisites":["` + prereq + `"]}`
	}
	tooFew := `{"title":"t","concepts":[` + concept("a", seed) + `]}`
	good := `{"title":"새 챕터","summary":"요약","concepts":[` + strings.Join([]string{
		concept("a", seed), concept("b", "a"), concept("c", "b"), concept("d", "c"), concept("e", "d")}, ",") + `]}`
	fake := &sequenceLLM{texts: []string{tooFew, good}}
	s.LLM = fake
	res, err := s.CreateChapterDraft(ctx, user, ChapterRequest{FreeText: "운영체제 더 깊게"})
	if err != nil {
		t.Fatalf("draft: %v", err)
	}
	if len(fake.users) != 2 || !strings.Contains(fake.users[1], "이전 설계의 문제") {
		t.Fatalf("expected one repair call with the problems: %d calls", len(fake.users))
	}
	if res.DraftID == "" || len(res.Draft.Concepts) != 5 || res.Draft.Concepts[0].ID != chapter.OwnerPrefix(user)+".a" {
		t.Fatalf("draft = %+v", res)
	}
	if _, err := s.CreateChapterDraft(ctx, user, ChapterRequest{}); !errors.Is(err, ErrInvalidChapterRequest) {
		t.Fatalf("empty request = %v", err)
	}
	s.LLM = &sequenceLLM{texts: []string{tooFew}}
	var design *ChapterDesignError
	if _, err := s.CreateChapterDraft(ctx, user, ChapterRequest{FreeText: "x"}); !errors.As(err, &design) || len(design.Problems) == 0 {
		t.Fatalf("persistently invalid design = %v", err)
	}
}
