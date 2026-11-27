package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/internal/services"
	"github.com/forgen-team/punisher/internal/store"
	"github.com/forgen-team/punisher/migrations"
	"github.com/go-chi/chi/v5"
)

// threadsLLM: resolver JSON for the deep lecture, QA JSON, else a passage explanation.
type threadsLLM struct{ resolveTo string }

func (f *threadsLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	switch {
	case strings.Contains(req.User, "## 후보 개념"):
		return llm.MessagesResult{Text: `{"concept_ids":["` + f.resolveTo + `"],"confidence":0.9,"proposed":null}`, Model: "fake"}, nil
	case strings.Contains(req.User, "JSON만 출력"):
		return llm.MessagesResult{Text: `{"answer":"QA 답","linked_concept_ids":[],"proposed_concepts":[]}`, Model: "fake"}, nil
	}
	return llm.MessagesResult{Text: "구절 설명", Model: "fake"}, nil
}

func (f *threadsLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return f.Create(ctx, key, req)
}

type threadsFixture struct {
	h         http.Handler
	st        *store.SQLiteStore
	lectureID string
	childID   string
	svc       *services.Service
}

func newThreadsFixture(t *testing.T) threadsFixture {
	t.Helper()
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "punisher.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	if err := st.Migrate(ctx, migrations.Files); err != nil {
		t.Fatal(err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := st.SyncCurriculum(ctx, seed); err != nil {
		t.Fatal(err)
	}
	lecture := func(suffix string) string {
		id := "test.thread-" + suffix
		if _, err := st.DB().ExecContext(ctx, `INSERT INTO concepts (id, name, area, depth, summary, owner_user_id)
			VALUES (?, ?, 'os', 1, '요약', 'local')`, id, "스레드 "+suffix); err != nil {
			t.Fatal(err)
		}
		if _, err := st.DB().ExecContext(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '목표', 1, 1)`, id+"#1", id); err != nil {
			t.Fatal(err)
		}
		lec, err := st.InsertLecture(ctx, store.NewLecture{ConceptID: id, ObjectiveIDs: []string{id + "#1"},
			ObjectiveKey: learning.ObjectiveKey([]string{id + "#1"}), Depth: 1, Kind: learning.JobInstant, Status: "published",
			Title: "강의 " + suffix, Content: []byte(`{"title":"t","sections":[{"anchor":"s1","title":"섹션","blocks":[{"type":"p","text":"본문 문장"}]}]}`),
			Sources: []byte(`[]`), RequestedBy: "local", Language: learning.LocaleKO})
		if err != nil {
			t.Fatal(err)
		}
		return lec.ID
	}
	parent, child := lecture("a"), lecture("b")
	svc := &services.Service{Store: st, Keys: services.GatewayKeys{}, Hub: services.NewHub(), LLM: &threadsLLM{resolveTo: "test.thread-b"}}
	learn := &LearnHandler{Store: st, Svc: svc}
	r := chi.NewRouter()
	r.Get("/v1/me/settings", learn.GetSettings)
	r.Put("/v1/me/settings", learn.PutSettings)
	r.Post("/v1/me/level-suggestion/dismiss", learn.DismissLevelSuggestion)
	r.Get("/v1/lectures/{id}/questions", learn.ListQuestions)
	r.Post("/v1/lectures/{id}/threads", learn.CreateThread)
	r.Post("/v1/questions/{id}/followups", learn.CreateQuestionFollowup)
	r.Post("/v1/questions/{id}/deep-lecture", learn.DeepLecture)
	return threadsFixture{h: r, st: st, lectureID: parent, childID: child, svc: svc}
}

func send(t *testing.T, h http.Handler, method, url, body string, wantCode int, out any) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, url, strings.NewReader(body)))
	if rec.Code != wantCode {
		t.Fatalf("%s %s %s: status %d: %s", method, url, body, rec.Code, rec.Body.String())
	}
	if out != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), out); err != nil {
			t.Fatal(err)
		}
	}
}

type questionJSON struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Answer   string `json:"answer"`
	Status   string `json:"status"`
	Quote    string `json:"quote"`
	ParentID string `json:"parentId"`
	Mode     string `json:"mode"`
	Anchor   string `json:"anchor"`
}

// Threads end to end: each root mode, follow-ups (parentId is the root), the
// flat list with the new fields, and the deep lecture in the instant shape.
func TestThreadRoutes(t *testing.T) {
	f := newThreadsFixture(t)
	threads := "/v1/lectures/" + f.lectureID + "/threads"
	var roots []questionJSON
	for _, body := range []string{
		`{"anchor":"s1","quote":"본문 문장","mode":"easier"}`,
		`{"anchor":"s1","quote":"본문 문장","mode":"deeper"}`,
		`{"anchor":"s1","quote":"본문 문장","mode":"question","question":"왜요?"}`,
	} {
		var out struct{ Question questionJSON }
		send(t, f.h, http.MethodPost, threads, body, http.StatusCreated, &out)
		if out.Question.Status != "answered" || out.Question.Quote != "본문 문장" || out.Question.ParentID != "" {
			t.Fatalf("root %s = %+v", body, out.Question)
		}
		roots = append(roots, out.Question)
	}
	if roots[0].Mode != "easier" || roots[0].Answer != "구절 설명" || roots[2].Mode != "question" || roots[2].Answer != "QA 답" {
		t.Fatalf("roots = %+v", roots)
	}

	var f1, f2 struct{ Question questionJSON }
	send(t, f.h, http.MethodPost, "/v1/questions/"+roots[2].ID+"/followups", `{"question":"그럼 이건?"}`, http.StatusCreated, &f1)
	send(t, f.h, http.MethodPost, "/v1/questions/"+f1.Question.ID+"/followups", `{"mode":"deeper"}`, http.StatusCreated, &f2)
	if f1.Question.ParentID != roots[2].ID || f1.Question.Mode != "question" || f2.Question.ParentID != roots[2].ID ||
		f2.Question.Mode != "deeper" || f2.Question.Anchor != "s1" || f2.Question.Quote != "" {
		t.Fatalf("followups = %+v / %+v", f1.Question, f2.Question)
	}

	var list struct{ Questions []map[string]any }
	send(t, f.h, http.MethodGet, "/v1/lectures/"+f.lectureID+"/questions", "", http.StatusOK, &list)
	if len(list.Questions) != 5 {
		t.Fatalf("list = %d items", len(list.Questions))
	}
	for _, q := range list.Questions {
		for _, key := range []string{"quote", "parentId", "mode"} {
			if _, ok := q[key]; !ok {
				t.Errorf("list item lacks %q: %v", key, q)
			}
		}
	}

	var deep struct {
		Status    string `json:"status"`
		ConceptID string `json:"conceptId"`
		LectureID string `json:"lectureId"`
	}
	send(t, f.h, http.MethodPost, "/v1/questions/"+f2.Question.ID+"/deep-lecture", "", http.StatusOK, &deep)
	if deep.Status != services.InstantExisting || deep.LectureID != f.childID || deep.ConceptID != "test.thread-b" {
		t.Fatalf("deep lecture = %+v", deep)
	}
	_, children, err := f.st.LectureLinks(context.Background(), "local", f.lectureID)
	if err != nil || len(children) != 1 || children[0].ID != f.childID {
		t.Fatalf("deep lecture must link under the thread's lecture: %+v (%v)", children, err)
	}
}

// 400 codes before any LLM work, 404 for unknown lectures and questions.
func TestThreadRouteErrors(t *testing.T) {
	f := newThreadsFixture(t)
	threads := "/v1/lectures/" + f.lectureID + "/threads"
	for body, want := range map[string]ErrorCode{
		`{"anchor":"s1","quote":"본문 문장","mode":"simpler"}`:                                                    ErrorInvalidThreadMode,
		`{"anchor":"s1","quote":"본문 문장","mode":"question"}`:                                                   ErrorQuestionRequired,
		`{"anchor":"s1","quote":"본문 문장"}`:                                                                     ErrorQuestionRequired,
		`{"anchor":"s1","quote":"","mode":"easier"}`:                                                          ErrorInvalidQuote,
		`{"anchor":"s1","quote":"` + strings.Repeat("가", services.MaxSelectionRunes+1) + `","mode":"easier"}`: ErrorInvalidQuote,
		`{"anchor":"s1","quote":"본문 문장","mode":"easier","x":1}`:                                               ErrorBadRequest,
	} {
		var got errorEnvelope
		send(t, f.h, http.MethodPost, threads, body, http.StatusBadRequest, &got)
		if got.Error.Code != want {
			t.Errorf("%.60s → %s, want %s", body, got.Error.Code, want)
		}
	}
	var root struct{ Question questionJSON }
	send(t, f.h, http.MethodPost, threads, `{"anchor":"s1","quote":"본문 문장","mode":"easier"}`, http.StatusCreated, &root)
	var got errorEnvelope
	send(t, f.h, http.MethodPost, "/v1/questions/"+root.Question.ID+"/followups", `{"mode":"question"}`, http.StatusBadRequest, &got)
	if got.Error.Code != ErrorQuestionRequired {
		t.Errorf("followup without question: %s", got.Error.Code)
	}
	send(t, f.h, http.MethodPost, "/v1/lectures/no-such-lecture/threads", `{"anchor":"s1","quote":"본문 문장","mode":"easier"}`, http.StatusNotFound, nil)
	send(t, f.h, http.MethodPost, "/v1/questions/no-such-question/followups", `{"mode":"easier"}`, http.StatusNotFound, nil)
	send(t, f.h, http.MethodPost, "/v1/questions/no-such-question/deep-lecture", "", http.StatusNotFound, nil)
}

// GET/PUT settings carry levelSuggestion (null until 5 easier requests at a
// lowerable level); dismiss hides it.
func TestSettingsLevelSuggestion(t *testing.T) {
	f := newThreadsFixture(t)
	type settingsJSON struct {
		LevelSuggestion *struct {
			To     string `json:"to"`
			Reason string `json:"reason"`
		} `json:"levelSuggestion"`
	}
	var raw map[string]json.RawMessage
	send(t, f.h, http.MethodGet, "/v1/me/settings", "", http.StatusOK, &raw)
	if string(raw["levelSuggestion"]) != "null" {
		t.Fatalf("levelSuggestion must be null at first: %s", raw["levelSuggestion"])
	}
	var put settingsJSON
	send(t, f.h, http.MethodPut, "/v1/me/settings", `{"readerLevel":"novice"}`, http.StatusOK, &put)
	threads := "/v1/lectures/" + f.lectureID + "/threads"
	for i := range learning.LevelSuggestionMinEasier {
		body := fmt.Sprintf(`{"anchor":"s1","quote":"본문 문장 %d","mode":"easier"}`, i)
		send(t, f.h, http.MethodPost, threads, body, http.StatusCreated, nil)
	}
	var got settingsJSON
	send(t, f.h, http.MethodGet, "/v1/me/settings", "", http.StatusOK, &got)
	if got.LevelSuggestion == nil || got.LevelSuggestion.To != learning.ReaderBeginner || got.LevelSuggestion.Reason != "easier_often" {
		t.Fatalf("novice with 5 easier = %+v", got.LevelSuggestion)
	}
	send(t, f.h, http.MethodPost, "/v1/me/level-suggestion/dismiss", "", http.StatusNoContent, nil)
	got = settingsJSON{}
	send(t, f.h, http.MethodGet, "/v1/me/settings", "", http.StatusOK, &got)
	if got.LevelSuggestion != nil {
		t.Fatalf("dismissed suggestion = %+v", got.LevelSuggestion)
	}
}
