package services

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// systemLLM records every request and answers with a fixed text.
type systemLLM struct {
	mu   sync.Mutex
	text string
	reqs []llm.MessagesRequest
}

func (f *systemLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	return llm.MessagesResult{Text: f.text, Model: "fake"}, nil
}

func (f *systemLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return f.Create(ctx, key, req)
}

func (f *systemLLM) last(t *testing.T) llm.MessagesRequest {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.reqs) == 0 {
		t.Fatal("no LLM call")
	}
	return f.reqs[len(f.reqs)-1]
}

var englishRule = strings.TrimSpace(prompts.OutputLanguageRule(learning.LocaleEN))

func hasEnglishRule(system string) bool { return strings.Contains(system, englishRule) }

func setUserLocale(t *testing.T, s *Service, user, locale string) {
	t.Helper()
	if _, err := s.Store.UpdateLearningSettings(context.Background(), user, store.LearningSettingsPatch{Locale: &locale}); err != nil {
		t.Fatal(err)
	}
}

// englishConcept is a private concept of user with an English overlay.
func englishConcept(t *testing.T, s *Service, user string) string {
	t.Helper()
	ctx := context.Background()
	id := "test.i18n-out-" + user
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO concepts (id, name, area, depth, summary, owner_user_id, name_en, summary_en)
		VALUES (?, '캐시', 'os', 1, '요약', ?, 'Cache', 'Summary')`, id, user); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal, text_en)
		VALUES (?, ?, '읽기 경로', 1, 1, 'Read path')`, id+"#1", id); err != nil {
		t.Fatal(err)
	}
	return id
}

// A lecture-bound call (passage explanation) follows the lecture's language,
// not the user's: the English rule is there only for an English lecture.
func TestExplainFollowsLectureLanguage(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	fake := &systemLLM{text: "설명"}
	s.LLM = fake
	conceptID := englishConcept(t, s, user)
	lec := func(lang string) models.Lecture {
		return models.Lecture{ConceptID: conceptID, Title: "t", Language: lang,
			Content: []byte(`{"title":"t","sections":[{"anchor":"s1","title":"S","blocks":[]}]}`)}
	}
	cases := []struct {
		locale, lecture string
		rule            bool
	}{
		{learning.LocaleKO, learning.LocaleKO, false},
		{learning.LocaleKO, learning.LocaleEN, true},
		{learning.LocaleEN, learning.LocaleKO, false},
	}
	for _, c := range cases {
		setUserLocale(t, s, user, c.locale)
		if _, err := s.ExplainSelection(ctx, user, lec(c.lecture), "s1", "입력이 커질 때 비용"); err != nil {
			t.Fatal(err)
		}
		req := fake.last(t)
		if hasEnglishRule(req.System) != c.rule {
			t.Fatalf("user %s, lecture %s: rule=%v want %v", c.locale, c.lecture, !c.rule, c.rule)
		}
		if strings.Contains(req.System, "## 출력 언어") && !c.rule {
			t.Fatalf("ko call carries a language rule: %q", req.System)
		}
	}
}

// The term popover is in the screen language: an English user gets the
// English rule and the English concept name in the prompt.
func TestGlossUsesUserLocale(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	fake := &systemLLM{text: "A cache keeps copies close."}
	s.LLM = fake
	conceptID := englishConcept(t, s, user)
	setUserLocale(t, s, user, learning.LocaleEN)
	if _, cached, err := s.Gloss(ctx, user, conceptID); err != nil || cached {
		t.Fatalf("gloss: cached=%v %v", cached, err)
	}
	req := fake.last(t)
	if !hasEnglishRule(req.System) || !strings.Contains(req.User, "Cache") {
		t.Fatalf("en gloss request: system=%q user=%q", req.System, req.User)
	}
}

// Dispute regrades follow the item's (= lecture's) language.
func TestRegradeFollowsItemLanguage(t *testing.T) {
	for _, c := range []struct {
		lang string
		rule bool
	}{{learning.LocaleKO, false}, {learning.LocaleEN, true}} {
		fake := &systemLLM{text: "not json"}
		s := &Service{LLM: fake}
		item := store.CheckItem{Prompt: "q", Language: c.lang}
		item.Rubric.Must, item.Rubric.Answer = []string{"m"}, "a"
		_, _, _ = s.regrade(context.Background(), "k", item, "answer")
		if got := hasEnglishRule(fake.last(t).System); got != c.rule {
			t.Fatalf("%s regrade rule=%v", c.lang, got)
		}
	}
}

// The prompt preview shows the same output-language rule as the real call, and
// an English user's example target uses the English names.
func TestPreviewShowsOutputLanguage(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	ko, err := s.PreviewPrompt(ctx, user, prompts.BlockTeacherSystem, nil)
	if err != nil || hasEnglishRule(ko.System) {
		t.Fatalf("ko preview: rule=%v %v", hasEnglishRule(ko.System), err)
	}
	setUserLocale(t, s, user, learning.LocaleEN)
	en, err := s.PreviewPrompt(ctx, user, prompts.BlockTeacherSystem, nil)
	if err != nil || !hasEnglishRule(en.System) {
		t.Fatalf("en preview: rule=%v %v", hasEnglishRule(en.System), err)
	}
}

// An English user's new lecture: the target carries English names and
// objectives, and the teacher system prompt gets the English rule.
func TestInstantLectureEnglishRuleAndNames(t *testing.T) {
	s, user := dbService(t)
	ctx := context.Background()
	conceptID := englishConcept(t, s, user)
	setUserLocale(t, s, user, learning.LocaleEN)
	tgt, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if tgt.Prompt.ConceptName != "Cache" || tgt.Prompt.Summary != "Summary" || len(tgt.Prompt.Objectives) != 1 ||
		tgt.Prompt.Objectives[0].Text != "Read path" {
		t.Fatalf("en target context: %q %q %+v", tgt.Prompt.ConceptName, tgt.Prompt.Summary, tgt.Prompt.Objectives)
	}
	fake := &recordingLLM{text: lectureMarkupFor(tgt.Plan.ObjectiveIDs)}
	s.LLM = fake
	job, _, err := s.Store.CreateJob(ctx, store.NewJob{
		IdempotencyKey: "test-i18n-rule:" + user, Kind: "instant", UserID: user, ConceptID: conceptID,
		ObjectiveKey: "test-i18n-rule:" + user, Depth: tgt.Plan.Depth, Reason: "test", Language: tgt.Plan.Language,
	})
	if err != nil {
		t.Fatal(err)
	}
	s.runInstant("k", user, job.ID, tgt, func(Event) {})
	if len(fake.systems) == 0 || !hasEnglishRule(fake.systems[0]) {
		t.Fatalf("teacher system without the English rule: %q", fake.systems)
	}
}
