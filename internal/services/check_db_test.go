package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
	"github.com/wooo-jin/punisher/internal/llm"
)

// checkLLM answers by the system prompt (items, tutor, regrade) from queues;
// safe for the background runs.
type checkLLM struct {
	mu                    sync.Mutex
	items, tutor, regrade []string
	calls                 map[string]int
	users                 map[string][]string
}

func (f *checkLLM) Create(_ context.Context, _ string, req llm.MessagesRequest) (llm.MessagesResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	kind := "tutor"
	switch {
	case strings.Contains(req.System, "출제자"):
		kind = "items"
	case strings.Contains(req.System, "독립 채점자"):
		kind = "regrade"
	}
	queue := map[string]*[]string{"items": &f.items, "tutor": &f.tutor, "regrade": &f.regrade}[kind]
	f.calls[kind]++
	f.users[kind] = append(f.users[kind], req.User)
	if len(*queue) == 0 {
		return llm.MessagesResult{}, fmt.Errorf("no scripted %s reply", kind)
	}
	text := (*queue)[0]
	*queue = (*queue)[1:]
	return llm.MessagesResult{Text: text, Model: "fake-" + req.Model, Usage: llm.Usage{InputTokens: 10, OutputTokens: 5}}, nil
}

func (f *checkLLM) Stream(ctx context.Context, key string, req llm.MessagesRequest, _ func(string) error) (llm.MessagesResult, error) {
	return f.Create(ctx, key, req)
}

func (f *checkLLM) push(kind string, texts ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	q := map[string]*[]string{"items": &f.items, "tutor": &f.tutor, "regrade": &f.regrade}[kind]
	*q = append(*q, texts...)
}

func (f *checkLLM) count(kind string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[kind]
}

type checkFixture struct {
	conceptID, lectureID, objA, objB string
}

// newCheckFixture adds a concept of the user's own (chapter-style, invisible to
// others) with two objectives and an instant lecture covering both.
func newCheckFixture(t *testing.T, s *Service, user string) checkFixture {
	t.Helper()
	ctx := context.Background()
	db := s.Store.(*store.SQLiteStore).DB() // SQLite port of the Pool() queries below
	f := checkFixture{conceptID: "test.mastery-" + strings.TrimPrefix(user, "test-prompts-")}
	f.objA, f.objB = f.conceptID+"#1", f.conceptID+"#2"
	content := `{"schemaVersion":1,"title":"페이지 캐시","sections":[
		{"anchor":"s1","kind":"principles","title":"원리","blocks":[{"anchor":"s1-b1","type":"markdown","text":"읽은 블록은 메모리에 남는다"}]},
		{"anchor":"s2","kind":"walkthrough","title":"따라가기","blocks":[{"anchor":"s2-b1","type":"markdown","text":"쓰기는 더티 페이지로 남는다"}]}]}`
	stmts := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES (?, '페이지 캐시', 'os', 1, ?)`, []any{f.conceptID, user}},
		{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '읽기 경로를 설명한다', 1, 1)`, []any{f.objA, f.conceptID}},
		{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '쓰기 지연을 설명한다', 1, 2)`, []any{f.objB, f.conceptID}},
	}
	for _, st := range stmts {
		if _, err := db.ExecContext(ctx, st.sql, st.args...); err != nil {
			t.Fatal(err)
		}
	}
	// The database is a temp file per test (dbService), so no cleanup is needed.
	f.lectureID = store.NewID()
	if _, err := db.ExecContext(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, title, content, requested_by)
		VALUES (?, ?, ?, ?, 1, 'instant', 'published', '페이지 캐시', ?, ?)`,
		f.lectureID, f.conceptID, `["`+f.objA+`","`+f.objB+`"]`, f.objA+","+f.objB, content, user); err != nil {
		t.Fatal(err)
	}
	return f
}

const leakAnswer = "읽은 블록은 페이지 캐시에 남아 다음 읽기는 디스크 대신 메모리에서 처리된다"

func itemJSON(obj, purpose, format, prompt, anchor string) string {
	return fmt.Sprintf(`{"objective_id":%q,"purpose":%q,"format":%q,"prompt":%q,"rubric":{"must":["메모리에서 읽는다"],"nice":[],"misconceptions":["항상 디스크"],"answer":%q,"anchor":%q}}`,
		obj, purpose, format, prompt, leakAnswer, anchor)
}

func itemSetJSON(f checkFixture, withIntegrative bool) string {
	items := []string{
		itemJSON(f.objA, "check", "recall", "같은 파일을 두 번째 읽으면 왜 빨라지나요?", "s1"),
		itemJSON(f.objB, "check", "predict", "write 직후 전원이 꺼지면 어떻게 될까요?", "s2"),
		itemJSON(f.objA, "review", "apply", "로그를 다시 읽는 배치가 왜 빠른지 설명해 보세요.", "s1-b1"),
		itemJSON(f.objB, "review", "recall", "더티 페이지는 언제 디스크로 가나요?", "s2-b1"),
	}
	if withIntegrative {
		items = append(items, itemJSON("", "check", "integrative", "읽기와 쓰기 경로를 함께 써서 DB 체크포인트를 설명해 보세요.", "s2"))
	}
	return `{"items":[` + strings.Join(items, ",") + `]}`
}

func tutorJSON(verdict, move, text string, confidence float64) string {
	return fmt.Sprintf(`{"verdict":%q,"met":["메모리"],"missing":["디스크 접근 감소"],"quote":"메모리","misconception":"","confidence":%v,"tutor_move":%q,"tutor_text":%q}`,
		verdict, confidence, move, text)
}

// waitIdle polls the session until nothing is running (the client's refetch loop).
func waitIdle(t *testing.T, s *Service, user, id string) models.CheckDetail {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		d, err := s.GetCheck(context.Background(), user, id)
		if err != nil {
			t.Fatal(err)
		}
		if !d.Pending {
			return d
		}
		if time.Now().After(deadline) {
			t.Fatalf("session still pending: %+v", d.Session)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func masteryOf(t *testing.T, s *Service, user, conceptID, objectiveID string) models.ObjectiveMastery {
	t.Helper()
	m, err := s.Mastery(context.Background(), user, conceptID)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range m.Objectives {
		if o.ObjectiveID == objectiveID {
			return o
		}
	}
	t.Fatalf("objective %s missing from %+v", objectiveID, m)
	return models.ObjectiveMastery{}
}

func answer(t *testing.T, s *Service, user, id, text string) models.CheckDetail {
	t.Helper()
	if _, err := s.AnswerCheck(context.Background(), user, id, text); err != nil {
		t.Fatalf("answer %q: %v", text, err)
	}
	return waitIdle(t, s, user, id)
}

func lastTurn(d models.CheckDetail) models.CheckTurn { return d.Turns[len(d.Turns)-1] }

// lastJudged is the latest turn carrying a verdict (an ended item is followed by the next ask).
func lastJudged(d models.CheckDetail) models.CheckTurn {
	for i := len(d.Turns) - 1; i >= 0; i-- {
		if d.Turns[i].Verdict != "" {
			return d.Turns[i]
		}
	}
	return models.CheckTurn{}
}

func checkService(t *testing.T) (*Service, *checkLLM, string, checkFixture) {
	s, user := dbService(t)
	fake := &checkLLM{calls: map[string]int{}, users: map[string][]string{}}
	s.LLM = fake
	s.Cfg = Config{LectureModel: "lecture", FastModel: "fast"}
	return s, fake, user, newCheckFixture(t, s, user)
}

// TestCheckSessionFlow: start → items (one rewrite) → correct / narrow+explain →
// immediate evidence → dispute restores → resume → integrative challenge → finished.
func TestCheckSessionFlow(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, false), itemSetJSON(f, true))

	d, created, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil || !created {
		t.Fatal(created, err)
	}
	id := d.Session.ID
	d = waitIdle(t, s, user, id)
	if d.Session.Status != "active" || len(d.Items) != 3 || d.Items[2].ObjectiveID != "" || lastTurn(d).Move != "ask" || !d.Current.CanAnswer || !d.Current.CanReveal {
		t.Fatalf("activated session = %+v current=%+v", d.Session, d.Current)
	}
	if d.Items[0].Answer != "" {
		t.Fatal("the model answer must stay hidden until the item is done")
	}
	if fake.count("items") != 2 || !strings.Contains(fake.users["items"][1], "통합 문항") {
		t.Fatal("the invalid set must be rewritten once with its problems")
	}
	var jobStatus string
	var attempts, tokens int
	if err := s.Store.(*store.SQLiteStore).DB().QueryRowContext(ctx, `SELECT status, json_array_length(attempt_log), input_tokens + output_tokens FROM generation_jobs
		WHERE user_id = ? AND kind = 'check_items'`, user).Scan(&jobStatus, &attempts, &tokens); err != nil || jobStatus != "succeeded" || attempts != 2 || tokens != 30 {
		t.Fatalf("items job = %s attempts=%d tokens=%d (%v)", jobStatus, attempts, tokens, err)
	}

	// 1번(목표 A): 첫 답 정답 → unaided_correct, 즉시 반영(box 1, 레벨 3).
	fake.push("tutor", tutorJSON("correct", "next", "맞아요, 메모리에서 읽습니다.", 0.9))
	d = answer(t, s, user, id, "메모리에서 읽어서요")
	if d.Session.Cursor != 1 || !d.Items[0].Done || d.Items[0].Outcome != learning.OutcomeUnaidedCorrect || d.Items[0].Answer == "" {
		t.Fatalf("after item 1: cursor=%d item=%+v", d.Session.Cursor, d.Items[0])
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	if a.Box != 1 || a.Best != learning.BestUnaided || a.Level != learning.LevelAlone || a.NextReviewAt == nil || a.LastGrader != "llm" {
		t.Fatalf("objective A mastery = %+v", a)
	}

	// 2번(목표 B): 부분 → 좁힌 질문 → 틀림 → 설명+재질문 → 틀림 → failed.
	fake.push("tutor", tutorJSON("partial", "narrow", "디스크에는 언제 쓰일까요?", 0.9))
	d = answer(t, s, user, id, "메모리에 남아요")
	if lastTurn(d).Move != "narrow" || !d.Current.CanReveal || d.Current.Answers != 1 {
		t.Fatalf("narrow: %+v current=%+v", lastTurn(d), d.Current)
	}
	fake.push("tutor", tutorJSON("incorrect", "explain", "쓰기는 더티 페이지로 남았다가 나중에 기록됩니다. 그러면 fsync 는 무엇을 보장할까요?", 0.9))
	d = answer(t, s, user, id, "바로 써져요")
	if lastTurn(d).Move != "explain" || d.Current.CanReveal || !lastTurn(d).Aided {
		t.Fatalf("explain: %+v current=%+v", lastTurn(d), d.Current)
	}
	if !strings.Contains(fake.users["tutor"][2], "narrow") || !strings.Contains(fake.users["tutor"][2], "explain | next") {
		t.Fatal("the tutor must get the dialogue and the allowed moves")
	}
	fake.push("tutor", tutorJSON("incorrect", "next", "fsync 는 기록을 기다립니다.", 0.9))
	d = answer(t, s, user, id, "모르겠어요")
	judgedB := lastJudged(d)
	if judgedB.Outcome != learning.OutcomeFailed || d.Session.Cursor != 2 || judgedB.Grade == nil || judgedB.Grade.Quote != "" {
		t.Fatalf("item 2 judgement = %+v (quote not in the answer is dropped)", judgedB)
	}
	b := masteryOf(t, s, user, f.conceptID, f.objB)
	if b.Best != learning.BestRevealed || b.Level != learning.LevelHeard || b.Box != 0 {
		t.Fatalf("objective B after failure = %+v", b)
	}

	// 이의 제기: 다른 모델이 첫 답을 정답으로 → 반영 전 상태로 복원 + best unaided.
	fake.push("regrade", `{"verdict":"correct","met":["메모리"],"missing":[],"quote":"메모리에 남아요","confidence":0.8}`)
	if _, err := s.DisputeCheck(ctx, user, id, judgedB.ID, "핵심은 말했어요"); err != nil {
		t.Fatal(err)
	}
	d = waitIdle(t, s, user, id)
	disp := lastTurn(d)
	if disp.Move != "dispute" || disp.Grader != learning.GraderRegrade || disp.Grade == nil || disp.Grade.Accepted == nil || !*disp.Grade.Accepted || disp.Model != "fake-fast" {
		t.Fatalf("dispute turn = %+v", disp)
	}
	if !strings.Contains(fake.users["regrade"][0], "메모리에 남아요") || strings.Contains(fake.users["regrade"][0], "핵심은 말했어요") {
		t.Fatal("the regrade sees the first answer only, never the learner's reason")
	}
	b = masteryOf(t, s, user, f.conceptID, f.objB)
	if b.Best != learning.BestUnaided || b.Box != 0 || b.Level != learning.LevelAlone || b.LastGrader != learning.GraderRegrade {
		t.Fatalf("objective B after dispute = %+v (restored, best unaided, box not raised)", b)
	}
	if _, err := s.DisputeCheck(ctx, user, id, judgedB.ID, ""); err == nil {
		t.Fatal("one dispute per item")
	}

	// 중단 후 재개: 다시 '확인하기' → 같은 세션, 같은 위치.
	again, created, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil || created || again.Session.ID != id || again.Session.Cursor != 2 {
		t.Fatalf("resume = %+v created=%v err=%v", again.Session, created, err)
	}

	// 3번(통합): 정답 → 반론 → 방어 → wrap, covers 에 integrative_ok, 세션 종료.
	fake.push("tutor", tutorJSON("correct", "challenge", "체크포인트 없이 WAL 만으로 충분하지 않나요?", 0.9),
		tutorJSON("correct", "wrap", "반론을 잘 막았습니다.", 0.9))
	d = answer(t, s, user, id, "읽기는 캐시, 쓰기는 더티 페이지라 체크포인트가 필요해요")
	if lastTurn(d).Move != "challenge" {
		t.Fatalf("integrative correct must be challenged: %+v", lastTurn(d))
	}
	d = answer(t, s, user, id, "WAL 이 길어지면 복구가 느려져요")
	if d.Session.Status != "finished" || d.Current != nil || lastTurn(d).Move != "wrap" || lastTurn(d).Outcome != learning.OutcomeUnaidedCorrect {
		t.Fatalf("finished = %+v last=%+v", d.Session, lastTurn(d))
	}
	if a = masteryOf(t, s, user, f.conceptID, f.objA); !a.IntegrativeOK || a.Box != 1 {
		t.Fatalf("integrative credit must not move the box: %+v", a)
	}
	var result struct {
		Items []store.ResultEntry `json:"items"`
	}
	if err := json.Unmarshal(d.Session.Result, &result); err != nil || len(result.Items) != 3 || result.Items[1].LevelBefore != 0 || result.Items[1].Anchor != "s2" {
		t.Fatalf("result = %s (%v)", d.Session.Result, err)
	}
	if lec, tok, err := s.Store.UsageSince(ctx, user, time.Now().Add(-time.Hour)); err != nil || lec != 0 || tok != 0 {
		t.Fatalf("check usage must not count toward the lecture budget: %d %d %v", lec, tok, err)
	}
	if tok, err := s.Store.UsageSinceChecks(ctx, user, time.Now().Add(-time.Hour)); err != nil || tok != 15*9 {
		t.Fatalf("check usage = %d (%v), want 9 calls × 15", tok, err)
	}
}

// TestCheckRevealHeldAbandonAndSweep: a cached set (no generation), held and
// revealed items, abandoning mid-item keeps evidence, sweep ends stale work.
func TestCheckRevealHeldAbandonAndSweep(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, true))
	first, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	waitIdle(t, s, user, first.Session.ID)
	if _, err := s.AbandonCheck(ctx, user, first.Session.ID); err != nil {
		t.Fatal(err)
	}

	d, created, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil || !created || d.Session.Status != "active" || fake.count("items") != 1 {
		t.Fatalf("a new session reuses the cached set at once: %+v created=%v items calls=%d err=%v", d.Session, created, fake.count("items"), err)
	}
	id := d.Session.ID

	// 1번: 공개 → failed(revealed), 근거는 남는다.
	d, err = s.RevealCheck(ctx, user, id)
	if err != nil || len(d.Turns) < 2 {
		t.Fatal(err)
	}
	if rev := d.Turns[len(d.Turns)-2]; rev.Move != "reveal" || rev.Outcome != learning.OutcomeFailed || rev.Text != leakAnswer ||
		d.Session.Cursor != 1 || lastTurn(d).Move != "ask" {
		t.Fatalf("reveal = %+v then %+v", rev, lastTurn(d))
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); a.Best != learning.BestRevealed || a.Level != learning.LevelHeard {
		t.Fatalf("revealed mastery = %+v", a)
	}
	// 2번: 낮은 confidence → 보류(held), 숙달 변화 없음, 모범답 표시.
	fake.push("tutor", tutorJSON("correct", "next", "맞아요", 0.3))
	d = answer(t, s, user, id, "더티 페이지가 남아요")
	if h := lastJudged(d); h.Verdict != "held" || h.Outcome != learning.OutcomeHeld || !strings.Contains(h.Text, "판정을 보류") {
		t.Fatalf("held = %+v", h)
	}
	if b := masteryOf(t, s, user, f.conceptID, f.objB); b.UpdatedAt != nil || b.Level != 0 {
		t.Fatalf("held must not create evidence: %+v", b)
	}
	// 3번: 부분 → 좁힌 질문, 답하지 않고 그만하기 → unaided_partial 로 닫고 abandoned.
	fake.push("tutor", tutorJSON("partial", "narrow", "체크포인트가 없으면 무엇이 길어질까요?", 0.9))
	d = answer(t, s, user, id, "캐시 때문이에요")
	if lastTurn(d).Move != "narrow" {
		t.Fatalf("narrow = %+v", lastTurn(d))
	}
	d, err = s.AbandonCheck(ctx, user, id)
	if err != nil || d.Session.Status != "abandoned" || lastTurn(d).Outcome != learning.OutcomeUnaidedPartial || !d.Items[2].Done {
		t.Fatalf("abandon mid-item = %+v last=%+v (%v)", d.Session, lastTurn(d), err)
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); a.Best != learning.BestRevealed {
		t.Fatalf("abandoning keeps applied evidence: %+v", a)
	}
	sweepFixture(t, s, user, f)
}

// sweepFixture: a fresh session with a pending turn (interrupted run) is failed
// by the boot sweep and runs again on retry; idle sessions are abandoned.
func sweepFixture(t *testing.T, s *Service, user string, f checkFixture) {
	ctx := context.Background()
	db := s.Store.(*store.SQLiteStore).DB()
	if _, err := db.ExecContext(ctx, `UPDATE check_sessions SET status = 'abandoned' WHERE user_id = ? AND status = 'active'`, user); err != nil {
		t.Fatal(err)
	}
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	id := d.Session.ID
	tutor := &store.TurnWrite{ItemID: d.Items[0].ID, Role: "tutor", Status: "pending", Grader: "llm"}
	learner := &store.TurnWrite{ItemID: d.Items[0].ID, Role: "learner", Move: "answer", Text: "메모리"}
	if _, err := s.Store.StartCheckTurns(ctx, user, id, learner, tutor, 3, true); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AnswerCheck(ctx, user, id, "또 답"); err == nil {
		t.Fatal("no answer while a turn is pending")
	}
	if n, err := s.Store.SweepStaleChecksFor(ctx, user, 0, 0, AbandonCheckAge); err != nil || n < 1 {
		t.Fatalf("boot sweep = %d %v", n, err)
	}
	d = waitIdle(t, s, user, id)
	if lastTurn(d).Status != "failed" || d.Current.CanAnswer {
		t.Fatalf("swept turn = %+v current=%+v", lastTurn(d), d.Current)
	}
	s.LLM.(*checkLLM).push("tutor", tutorJSON("correct", "next", "맞아요.", 0.9))
	if _, err := s.RetryCheck(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	if d = waitIdle(t, s, user, id); d.Session.Cursor != 1 || !d.Items[0].Done {
		t.Fatalf("retried turn must grade the stored answer: %+v", d.Session)
	}
	if _, err := db.ExecContext(ctx, `UPDATE check_sessions SET updated_at = ? WHERE id = ?`, store.FormatTime(time.Now().Add(-8*24*time.Hour)), id); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Store.SweepStaleChecksFor(ctx, user, StaleCheckTurnAge, StaleCheckPrepareAge, AbandonCheckAge); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.Store.GetCheckSession(ctx, user, id); got.Status != "abandoned" || got.FinishedAt == nil {
		t.Fatalf("idle session = %+v", got)
	}
}

// TestCheckSelfGrade: a self-graded session (P2 reviews) records the answer
// without an LLM, opens the model answer, and the learner's grade ends the item.
func TestCheckSelfGrade(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	d = waitIdle(t, s, user, d.Session.ID)
	sess, _, err := s.Store.CreateCheckSession(ctx, store.NewCheckSession{UserID: user, Kind: "review", Grader: learning.GraderSelf,
		ConceptID: f.conceptID, SessionKey: "review:test"})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{d.Items[0].ID}
	if _, err := s.Store.ActivateCheckSession(ctx, user, sess.ID, ids, store.TurnWrite{ItemID: ids[0], Role: "tutor", Move: "ask", Text: d.Items[0].Prompt}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SelfGradeCheck(ctx, user, sess.ID, "correct"); err == nil {
		t.Fatal("self-grade needs an answer first")
	}
	got, err := s.AnswerCheck(ctx, user, sess.ID, "메모리에 남아요")
	if err != nil || got.Pending || got.Items[0].Answer == "" || !got.Current.CanSelfGrade || got.Current.CanAnswer {
		t.Fatalf("self answer = %+v current=%+v (%v)", got.Items, got.Current, err)
	}
	got, err = s.SelfGradeCheck(ctx, user, sess.ID, "partial")
	if err != nil || got.Session.Status != "finished" || lastJudged(got).Outcome != learning.OutcomeUnaidedPartial || lastJudged(got).Grader != "self" {
		t.Fatalf("self grade = %+v %+v (%v)", got.Session, lastJudged(got), err)
	}
	if fake.count("tutor") != 0 {
		t.Fatal("self grading makes no LLM call")
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); a.LastGrader != "self" || a.NextReviewAt == nil || a.Level != 0 {
		t.Fatalf("self-graded partial = %+v (box kept, next = due(0))", a)
	}
}
