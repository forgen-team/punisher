package services

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

// threadLLM answers the three lanes a thread touches: the resolver (JSON ids),
// the QA block (JSON answer) and the passage explanation (markdown).
type threadLLM struct {
	mu        sync.Mutex
	resolveTo string
	reqs      []llm.MessagesRequest
	// onCall, when set, runs inside the call (e.g. the client disconnecting);
	// a cancelled ctx then fails the call like a real provider would.
	onCall func()
}

func (f *threadLLM) Create(ctx context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reqs = append(f.reqs, req)
	if f.onCall != nil {
		f.onCall()
	}
	if err := ctx.Err(); err != nil {
		return llm.MessagesResult{}, err
	}
	switch {
	case strings.Contains(req.User, "## 후보 개념"):
		return llm.MessagesResult{Text: `{"concept_ids":["` + f.resolveTo + `"],"confidence":0.9,"proposed":null}`, Model: "fake"}, nil
	case strings.Contains(req.User, "JSON만 출력"):
		return llm.MessagesResult{Text: `{"answer":"QA 답","linked_concept_ids":[],"proposed_concepts":[]}`, Model: "fake"}, nil
	}
	return llm.MessagesResult{Text: "구절 설명", Model: "fake"}, nil
}

func (f *threadLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return f.Create(ctx, key, req)
}

func (f *threadLLM) calls() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.reqs)
}

func (f *threadLLM) last() llm.MessagesRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.reqs[len(f.reqs)-1]
}

// threadConcept is a private one-objective concept of user (no prerequisites).
func threadConcept(t *testing.T, s *Service, user, suffix string) string {
	t.Helper()
	ctx := context.Background()
	id := "test.thread-" + suffix
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO concepts (id, name, area, depth, summary, owner_user_id)
		VALUES (?, ?, 'os', 1, '요약', ?)`, id, "스레드 개념 "+suffix, user); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal)
		VALUES (?, ?, '목표', 1, 1)`, id+"#1", id); err != nil {
		t.Fatal(err)
	}
	return id
}

// threadLecture is a published instant lecture of user covering conceptID's plan.
func threadLecture(t *testing.T, s *Service, user, conceptID string) models.Lecture {
	t.Helper()
	ctx := context.Background()
	tgt, err := s.instantTarget(ctx, user, conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	lec, err := s.Store.InsertLecture(ctx, store.NewLecture{ConceptID: conceptID, ObjectiveIDs: tgt.Plan.ObjectiveIDs,
		ObjectiveKey: tgt.Plan.ObjectiveKey, Depth: tgt.Plan.Depth, Kind: learning.JobInstant, Status: "published",
		Title: "스레드 강의", Content: []byte(`{"title":"t","sections":[{"anchor":"s1","title":"섹션","blocks":[{"type":"p","text":"본문 문장"}]}]}`),
		Sources: []byte(`[]`), RequestedBy: user, Language: learning.LocaleKO})
	if err != nil {
		t.Fatal(err)
	}
	return lec
}

func threadService(t *testing.T) (*Service, string, *threadLLM, models.Lecture) {
	t.Helper()
	s, user := dbService(t)
	fake := &threadLLM{}
	s.LLM = fake
	return s, user, fake, threadLecture(t, s, user, threadConcept(t, s, user, "a"))
}

// Root easier/deeper use the passage-explanation block with the mode's task and
// the reader section; a question root uses the QA block with the quote.
func TestStartThreadModes(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	setReaderLevel(t, s, user, learning.ReaderNovice)
	reader := prompts.ReaderProfile(learning.ReaderNovice, learning.LocaleKO)
	for _, c := range []struct{ mode, question, answer, task string }{
		{learning.ThreadEasier, "", "구절 설명", "**더 쉽게**"},
		{learning.ThreadDeeper, "", "구절 설명", "**더 깊게**"},
		{learning.ThreadQuestion, "왜 그런가요?", "QA 답", "아래 질문에 직접 답하세요"},
	} {
		q, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: c.mode, Question: c.question})
		if err != nil || q.Status != "answered" || q.Answer != c.answer || q.Mode != c.mode || q.Quote != "본문 문장" || q.ParentID != "" {
			t.Fatalf("%s root = %+v (%v)", c.mode, q, err)
		}
		if c.question == "" && q.Question == "" {
			t.Errorf("%s root must store the request text", c.mode)
		}
		req := fake.last()
		for _, want := range []string{"본문 문장", c.task, "300~500자", reader} {
			if !strings.Contains(req.User, want) {
				t.Errorf("%s prompt lacks %q", c.mode, want)
			}
		}
		if req.Priority != llm.PriorityShort {
			t.Errorf("%s priority = %v, want the short lane", c.mode, req.Priority)
		}
	}
}

// Follow-ups always point at the root, carry the earlier turns, and
// easier/deeper re-explain the previous answer.
func TestFollowupThreadsToRoot(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	root, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadQuestion, Question: "첫 질문"})
	if err != nil {
		t.Fatal(err)
	}
	f1, err := s.Followup(ctx, user, root.ID, learning.ThreadQuestion, "둘째 질문")
	if err != nil || f1.ParentID != root.ID || f1.Anchor != "s1" || f1.Status != "answered" {
		t.Fatalf("followup 1 = %+v (%v)", f1, err)
	}
	f2, err := s.Followup(ctx, user, f1.ID, learning.ThreadEasier, "")
	if err != nil || f2.ParentID != root.ID || f2.Mode != learning.ThreadEasier {
		t.Fatalf("followup of a followup must point at the root: %+v (%v)", f2, err)
	}
	u := fake.last().User
	for _, want := range []string{"본문 문장", "Q: 첫 질문", "Q: 둘째 질문", "직전 답을 **더 쉽게**"} {
		if !strings.Contains(u, want) {
			t.Errorf("followup prompt lacks %q", want)
		}
	}
	if _, err := s.Followup(ctx, "someone-else", root.ID, learning.ThreadQuestion, "x"); err == nil {
		t.Error("another user's thread must not be found")
	}
	alt, err := s.Store.CreateQuestion(ctx, user, models.Question{LectureID: lec.ID, Kind: "alt_explanation", Question: "다르게"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Followup(ctx, user, alt.ID, learning.ThreadQuestion, "x"); err != ErrNotThread {
		t.Errorf("alt explanation follow-up: %v, want ErrNotThread", err)
	}
}

// The deep lecture query is the quote plus the reader's questions; the instant
// start reuses a published lecture of the resolved concept and links it under the thread's lecture.
func TestDeepLectureLinksUnderThreadLecture(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	childConcept := threadConcept(t, s, user, "b")
	child := threadLecture(t, s, user, childConcept)
	fake.resolveTo = childConcept
	root, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1-2", Quote: "본문\n  문장", Mode: learning.ThreadEasier})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Followup(ctx, user, root.ID, learning.ThreadQuestion, "더 알고 싶은 점"); err != nil {
		t.Fatal(err)
	}
	resp, err := s.DeepLecture(ctx, user, root.ID)
	if err != nil || resp.Status != InstantExisting || resp.LectureID != child.ID || resp.ConceptID != childConcept {
		t.Fatalf("deep lecture = %+v (%v)", resp, err)
	}
	if u := fake.reqs[len(fake.reqs)-1].User; !strings.Contains(u, `“본문 문장” — 더 알고 싶은 점`) {
		t.Errorf("resolver query lacks the thread summary:\n%s", u)
	}
	_, children, err := s.Store.LectureLinks(ctx, user, lec.ID)
	if err != nil || len(children) != 1 || children[0].ID != child.ID || children[0].Anchor != "s1-2" {
		t.Fatalf("children = %+v (%v)", children, err)
	}
}

// A thread about the lecture's own concept must not reopen the lecture being
// read: the deep lecture becomes a question explanation of the thread.
func TestDeepLectureOnSameConceptStartsInquiry(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	fake.resolveTo = lec.ConceptID
	root, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1-2", Quote: "같은 개념 문장", Mode: learning.ThreadEasier})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := s.DeepLecture(ctx, user, root.ID)
	if err != nil || resp.Status != InstantInquiry || resp.InquiryID == "" || resp.LectureID != "" {
		t.Fatalf("deep lecture = %+v (%v), want an inquiry", resp, err)
	}
	inq, err := s.Store.GetInquiry(ctx, user, resp.InquiryID)
	if err != nil || !strings.Contains(inq.Question, "같은 개념 문장") || !strings.Contains(inq.Question, lec.Title) {
		t.Fatalf("inquiry = %+v (%v)", inq, err)
	}
}

func TestDeepLectureQueryBounded(t *testing.T) {
	long := strings.Repeat("가", 400)
	q := DeepLectureQuery([]models.Question{
		{Quote: long, Mode: learning.ThreadEasier, Question: "더 쉽게 설명해 주세요."},
		{Mode: learning.ThreadQuestion, Question: long}, {Mode: learning.ThreadQuestion, Question: long},
		{Mode: learning.ThreadQuestion, Question: long},
	})
	if n := len([]rune(q)); n > DeepLectureQueryRunes || strings.Contains(q, "더 쉽게 설명해") {
		t.Fatalf("query %d runes: %q", n, q)
	}
}

// levelSuggestion: 5 easier requests in a week, one level lower, none for
// beginner/unset, hidden for a week after dismiss, counted after a level change.
func TestSuggestLevelThresholdsAndDismiss(t *testing.T) {
	s, user, _, lec := threadService(t)
	ctx := context.Background()
	now := time.Now()
	s.Now = func() time.Time { return now }
	suggest := func() *LevelSuggestion {
		t.Helper()
		st, err := s.Store.GetLearningSettings(ctx, user)
		if err != nil {
			t.Fatal(err)
		}
		got, err := s.SuggestLevel(ctx, user, st)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	easier := func(n int) {
		t.Helper()
		for range n {
			q, err := s.Store.CreateQuestion(ctx, user, models.Question{LectureID: lec.ID, Kind: "question",
				Question: "쉽게", Mode: learning.ThreadEasier})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := s.Store.AnswerQuestion(ctx, q.ID, "답", "fake", nil, nil); err != nil {
				t.Fatal(err)
			}
		}
	}
	easier(5)
	if got := suggest(); got != nil {
		t.Fatalf("unset level must not get a suggestion: %+v", got)
	}
	setReaderLevel(t, s, user, learning.ReaderAdvanced)
	if _, err := testDB(s).ExecContext(ctx, `UPDATE user_learning_settings SET reader_level_changed_at = ?`,
		store.FormatTime(now.Add(-8*24*time.Hour))); err != nil {
		t.Fatal(err)
	}
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lecture_questions SET created_at = ? WHERE id IN (SELECT id FROM lecture_questions LIMIT 1)`,
		store.FormatTime(now.Add(-8*24*time.Hour))); err != nil {
		t.Fatal(err)
	}
	if got := suggest(); got != nil {
		t.Fatalf("4 in the window must not suggest: %+v", got)
	}
	easier(1)
	if got := suggest(); got == nil || got.To != learning.ReaderPractitioner || got.Reason != "easier_often" {
		t.Fatalf("advanced with 5 easier = %+v", got)
	}
	if err := s.DismissLevelSuggestion(ctx, user); err != nil {
		t.Fatal(err)
	}
	if got := suggest(); got != nil {
		t.Fatalf("dismissed suggestion must hide: %+v", got)
	}
	now = now.Add(8 * 24 * time.Hour)
	easier(5)
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lecture_questions SET created_at = ?`, store.FormatTime(now.Add(-time.Hour))); err != nil {
		t.Fatal(err)
	}
	if got := suggest(); got == nil || got.To != learning.ReaderPractitioner {
		t.Fatalf("a week after dismiss the suggestion returns: %+v", got)
	}
	setReaderLevel(t, s, user, learning.ReaderBeginner)
	if got := suggest(); got != nil {
		t.Fatalf("beginner has no lower level: %+v", got)
	}
	for level, want := range map[string]string{learning.ReaderAdvanced: learning.ReaderPractitioner,
		learning.ReaderPractitioner: learning.ReaderNovice, learning.ReaderNovice: learning.ReaderBeginner,
		learning.ReaderBeginner: "", "": ""} {
		if got := learning.LowerReaderLevel(level); got != want {
			t.Errorf("lower(%q) = %q, want %q", level, got, want)
		}
	}
}

// A client that disconnects mid-answer cancels the LLM call, but the entry
// must still end failed (not stay pending forever): roots on both lanes and follow-ups.
func TestThreadCancelledRequestEndsFailed(t *testing.T) {
	s, user, fake, lec := threadService(t)
	bg := context.Background()
	root, err := s.StartThread(bg, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadQuestion, Question: "처음"})
	if err != nil || root.Status != "answered" {
		t.Fatalf("root = %+v (%v)", root, err)
	}
	for name, run := range map[string]func(context.Context) (models.Question, error){
		"question root": func(ctx context.Context) (models.Question, error) {
			return s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadQuestion, Question: "끊긴 질문"})
		},
		"easier root": func(ctx context.Context) (models.Question, error) {
			return s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadEasier})
		},
		"follow-up": func(ctx context.Context) (models.Question, error) {
			return s.Followup(ctx, user, root.ID, learning.ThreadQuestion, "끊긴 이어 묻기")
		},
	} {
		ctx, cancel := context.WithCancel(bg)
		fake.onCall = cancel
		q, err := run(ctx)
		fake.onCall = nil
		cancel()
		if err != nil || q.ID == "" || q.Status != "failed" {
			t.Fatalf("%s: returned %+v (%v), want the stored failed entry", name, q, err)
		}
		stored, err := s.Store.GetQuestion(bg, user, q.ID)
		if err != nil || stored.Status != "failed" || stored.Error == "" {
			t.Fatalf("%s: stored = %+v (%v), want failed with a reason", name, stored, err)
		}
	}
}

// A double tap within the window returns the first entry and makes one LLM call.
func TestThreadDoubleSubmitDeduped(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	req := ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadEasier}
	first, err := s.StartThread(ctx, user, lec, req)
	if err != nil || first.Status != "answered" {
		t.Fatalf("first = %+v (%v)", first, err)
	}
	calls := fake.calls()
	second, err := s.StartThread(ctx, user, lec, req)
	if err != nil || second.ID != first.ID || second.Answer != first.Answer || fake.calls() != calls {
		t.Fatalf("double root submit = %+v (%v), %d new LLM calls", second, err, fake.calls()-calls)
	}
	f1, err := s.Followup(ctx, user, first.ID, learning.ThreadQuestion, "왜요?")
	if err != nil || f1.Status != "answered" {
		t.Fatalf("followup = %+v (%v)", f1, err)
	}
	calls = fake.calls()
	f2, err := s.Followup(ctx, user, f1.ID, learning.ThreadQuestion, "왜요?") // 이어 묻기의 이어 묻기도 같은 루트 키
	if err != nil || f2.ID != f1.ID || fake.calls() != calls {
		t.Fatalf("double follow-up = %+v (%v), %d new LLM calls", f2, err, fake.calls()-calls)
	}
	thread, err := s.Store.ListThread(ctx, user, first.ID)
	if err != nil || len(thread) != 2 {
		t.Fatalf("thread has %d entries (%v), want root + one follow-up", len(thread), err)
	}
}

// Over the budget the thread call is refused before any entry exists.
func TestThreadBudgetCheckedBeforeEntry(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	root, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadEasier})
	if err != nil {
		t.Fatal(err)
	}
	s.recordUsage(ctx, user, "question", lec.ConceptID, usage{model: "fake", input: 100}, nil)
	s.Cfg.DailyTokenLimit = 50
	before, err := s.Store.ListQuestions(ctx, user, lec.ID)
	if err != nil {
		t.Fatal(err)
	}
	calls := fake.calls()
	if _, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "다른 문장", Mode: learning.ThreadDeeper}); !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("root over budget: %v, want ErrBudgetExceeded", err)
	}
	if _, err := s.Followup(ctx, user, root.ID, learning.ThreadQuestion, "더"); !errors.Is(err, ErrBudgetExceeded) {
		t.Errorf("follow-up over budget: %v, want ErrBudgetExceeded", err)
	}
	after, err := s.Store.ListQuestions(ctx, user, lec.ID)
	if err != nil || len(after) != len(before) || fake.calls() != calls {
		t.Fatalf("refused calls left %d entries / %d LLM calls", len(after)-len(before), fake.calls()-calls)
	}
}

// The deep-lecture inquiry question follows the lecture's language.
func TestDeepInquiryQuestionLanguage(t *testing.T) {
	ko := deepInquiryQuestion(models.Lecture{Title: "캐시", Language: learning.LocaleKO}, "“구절”")
	en := deepInquiryQuestion(models.Lecture{Title: "Caches", Language: learning.LocaleEN}, "“passage”")
	if !strings.HasPrefix(ko, "강의 「캐시」") || !strings.HasSuffix(ko, "“구절”") {
		t.Errorf("ko = %q", ko)
	}
	if !strings.HasPrefix(en, "I got stuck reading the lecture “Caches”") || strings.ContainsAny(en, "강의") {
		t.Errorf("en = %q", en)
	}
}

// The lecture-wide Ask history uses plain questions only — no thread roots,
// follow-ups or alt explanations.
func TestAskHistoryExcludesThreads(t *testing.T) {
	s, user, fake, lec := threadService(t)
	ctx := context.Background()
	if q, err := s.Ask(ctx, user, lec, "s1", "평범한 질문"); err != nil || q.Status != "answered" {
		t.Fatalf("ask = %+v (%v)", q, err)
	}
	root, err := s.StartThread(ctx, user, lec, ThreadRequest{Anchor: "s1", Quote: "본문 문장", Mode: learning.ThreadQuestion, Question: "스레드 루트 질문"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Followup(ctx, user, root.ID, learning.ThreadQuestion, "스레드 이어 묻기"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Ask(ctx, user, lec, "s1", "두 번째 질문"); err != nil {
		t.Fatal(err)
	}
	u := fake.last().User
	if !strings.Contains(u, "평범한 질문") {
		t.Errorf("history lacks the plain question:\n%s", u)
	}
	for _, leaked := range []string{"스레드 루트 질문", "스레드 이어 묻기"} {
		if strings.Contains(u, leaked) {
			t.Errorf("history must not include thread entry %q", leaked)
		}
	}
}
