package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/prompts"
	"github.com/forgen-team/punisher/internal/store"
)

func tutorWithMisconception(verdict, move, text, misconception string) string {
	return fmt.Sprintf(`{"verdict":%q,"met":[],"missing":["디스크 접근 감소"],"quote":"","misconception":%q,"confidence":0.9,"tutor_move":%q,"tutor_text":%q}`,
		verdict, misconception, move, text)
}

// extraConcept adds a concept of the user's own with one objective, the given
// state and (best ≠ "") a mastery row; edgeTo ≠ "" makes it a prerequisite of edgeTo.
func extraConcept(t *testing.T, s *Service, user, id, name, state, best string, lapses int, edgeTo string) string {
	t.Helper()
	ctx := context.Background()
	db := testDB(s)
	obj := id + "#1"
	type stmt struct {
		sql  string
		args []any
	}
	stmts := []stmt{
		{`INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES (?, ?, 'db', 1, ?)`, []any{id, name, user}},
		{`INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '목표', 1, 1)`, []any{obj, id}},
		{`INSERT INTO user_concept_states (user_id, concept_id, state) VALUES (?, ?, ?)`, []any{user, id, state}},
	}
	if best != "" {
		stmts = append(stmts, stmt{`INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, best, lapses) VALUES (?, ?, ?, ?, ?)`,
			[]any{user, obj, id, best, lapses}})
	}
	if edgeTo != "" {
		stmts = append(stmts, stmt{`INSERT INTO concept_edges (from_id, to_id, kind) VALUES (?, ?, 'prerequisite')`, []any{id, edgeTo}})
	}
	for _, st := range stmts {
		if _, err := db.ExecContext(ctx, st.sql, st.args...); err != nil {
			t.Fatal(err)
		}
	}
	return obj
}

func hasRef(refs []prompts.ConceptRef, id string) bool {
	for _, r := range refs {
		if r.ID == id {
			return true
		}
	}
	return false
}

// TestWeakConceptsMoveOutOfLearned: a completed concept with weak check evidence
// leaves "배운 개념" and Catalog ✓ for "다시 짚을 개념" in user-owned lectures and
// question explanations; library lectures keep the shared lists (M6).
func TestWeakConceptsMoveOutOfLearned(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	suffix := strings.TrimPrefix(user, "test-prompts-")
	weakPre, strongPre, weakOther := "test.weak-pre-"+suffix, "test.strong-pre-"+suffix, "test.weak-other-"+suffix
	extraConcept(t, s, user, weakPre, "약한 선행", "completed", learning.BestAided, 0, f.conceptID)
	extraConcept(t, s, user, strongPre, "확인 안 한 선행", "completed", "", 0, f.conceptID)
	extraConcept(t, s, user, weakOther, "약한 다른 개념", "completed", learning.BestRevealed, 0, "")

	inst, err := s.instantTarget(ctx, user, f.conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	pc := inst.LectureContext()
	if hasRef(pc.Learned, weakPre) || !hasRef(pc.Learned, strongPre) || !hasRef(pc.Revisit, weakPre) || !hasRef(pc.Revisit, weakOther) {
		t.Fatalf("instant: learned=%+v revisit=%+v", pc.Learned, pc.Revisit)
	}
	for _, c := range pc.Catalog {
		if c.ID == weakOther {
			t.Fatal("weak concept must leave the catalog (✓)")
		}
	}

	// 단독판: 라이브러리가 사용자 프롬프트를 써도(LibraryUsesUserPrompts) 개인화 블록은 넣지 않는다(docs/SYNC.md).
	for _, own := range []bool{false, true} {
		s.Cfg.LibraryUsesUserPrompts = own
		lib, err := s.libraryTarget(ctx, user, f.conceptID)
		if err != nil {
			t.Fatal(err)
		}
		lpc := lib.LectureContext()
		if !hasRef(lpc.Learned, weakPre) || len(lpc.Revisit) != 0 {
			t.Fatalf("library (user prompts %v) keeps the shared lists: learned=%+v revisit=%+v", own, lpc.Learned, lpc.Revisit)
		}
		if text := prompts.Outline(lpc, lib.Prompts); strings.Contains(text, "다시 짚을 개념") || !strings.Contains(text, weakOther+" 약한 다른 개념✓") {
			t.Fatalf("library prompt (user prompts %v) must not be personalized:\n%s", own, text)
		}
	}
	s.Cfg.LibraryUsesUserPrompts = false

	ic, _, err := s.inquiryContext(ctx, user, "왜 느려요?")
	if err != nil {
		t.Fatal(err)
	}
	if hasRef(ic.Learned, weakPre) || hasRef(ic.Learned, weakOther) || !hasRef(ic.Revisit, weakPre) || !hasRef(ic.Learned, strongPre) {
		t.Fatalf("inquiry: learned=%+v revisit=%+v", ic.Learned, ic.Revisit)
	}
	if !strings.Contains(prompts.Inquiry(ic), "## 다시 짚을 개념") {
		t.Fatal("inquiry prompt lists the concepts to revisit")
	}
}

// TestMisconceptionAcknowledgeAndResolve: '맞아요, 그렇게 알았어요' stores the
// turn's misconception once; it reaches user-owned lectures, Q&A and question
// explanations (never the library); an unaided correct answer on the same
// objective resolves it.
func TestMisconceptionAcknowledgeAndResolve(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	id := d.Session.ID
	waitIdle(t, s, user, id)
	const wrong = "두 번째 읽기도 항상 디스크에서 읽는다"
	fake.push("tutor", tutorJSON("partial", "narrow", "어디에 남을까요?", 0.9),
		tutorJSON("incorrect", "explain", "읽은 블록은 메모리에 남습니다. 그러면 두 번째 읽기는 어디서 올까요?", 0.9),
		tutorWithMisconception("incorrect", "next", "메모리에서 옵니다.", wrong))
	answer(t, s, user, id, "디스크에서요")
	answer(t, s, user, id, "여전히 디스크")
	d = answer(t, s, user, id, "그래도 디스크")
	judged := lastJudged(d)
	if judged.Grade == nil || judged.Grade.Misconception != wrong || judged.Outcome != learning.OutcomeFailed {
		t.Fatalf("judged = %+v", judged)
	}

	if _, _, err := s.AcknowledgeMisconception(ctx, user, ""); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("empty turnId → 400: %v", err)
	}
	if _, _, err := s.AcknowledgeMisconception(ctx, user, askTurn(d, d.Items[0].ID).ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("a turn without misconception → not found: %v", err)
	}
	if _, _, err := s.AcknowledgeMisconception(ctx, "test-other-user", judged.ID); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("another user's turn → not found: %v", err)
	}
	m, created, err := s.AcknowledgeMisconception(ctx, user, judged.ID)
	if err != nil || !created || m.Text != wrong || m.ObjectiveID != f.objA || m.ConceptID != f.conceptID || m.Status != learning.MisconceptionOpen {
		t.Fatalf("acknowledged = %+v created=%v (%v)", m, created, err)
	}
	if again, created, err := s.AcknowledgeMisconception(ctx, user, judged.ID); err != nil || created || again.ID != m.ID {
		t.Fatalf("second press returns the same row: %+v %v %v", again, created, err)
	}
	if d, err = s.GetCheck(ctx, user, id); err != nil || len(d.MisconceptionTurnIDs) != 1 || d.MisconceptionTurnIDs[0] != judged.ID {
		t.Fatalf("detail marks the acknowledged turn: %+v (%v)", d.MisconceptionTurnIDs, err)
	}
	if _, err := s.ListMisconceptions(ctx, user, "closed"); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("bad status → 400: %v", err)
	}
	if open, err := s.ListMisconceptions(ctx, user, ""); err != nil || len(open) != 1 || open[0].ObjectiveText == "" {
		t.Fatalf("open list = %+v (%v)", open, err)
	}

	inst, err := s.instantTarget(ctx, user, f.conceptID, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if text := prompts.Lecture(inst.LectureContext(), "", "", inst.Prompts); !strings.Contains(text, "## 이 수강생이 인정한 오개념") || !strings.Contains(text, wrong) {
		t.Fatal("user-owned lecture prompt carries the misconception")
	}
	for _, own := range []bool{false, true} { // 단독판: 사용자 프롬프트를 쓰는 라이브러리도 같다(docs/SYNC.md).
		s.Cfg.LibraryUsesUserPrompts = own
		lib, err := s.libraryTarget(ctx, user, f.conceptID)
		if err != nil {
			t.Fatal(err)
		}
		if text := prompts.Lecture(lib.LectureContext(), "", "", lib.Prompts); strings.Contains(text, wrong) || strings.Contains(text, "인정한 오개념") {
			t.Fatalf("library lecture prompt (user prompts %v) must not carry misconceptions (M6)", own)
		}
	}
	s.Cfg.LibraryUsesUserPrompts = false
	lec, err := s.Store.GetLecture(ctx, user, f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	qa, err := s.qaContext(ctx, user, lec, "s1", false)
	if err != nil || !strings.Contains(prompts.Question(qa.prompt, "왜요?"), wrong) {
		t.Fatalf("Q&A prompt carries the misconception (%v)", err)
	}
	ic, _, err := s.inquiryContext(ctx, user, "왜 느려요?")
	if err != nil || !strings.Contains(prompts.Inquiry(ic), wrong) {
		t.Fatalf("question explanation prompt carries the misconception (%v)", err)
	}
	pv, err := s.PreviewPrompt(ctx, user, prompts.BlockLectureFormat, nil)
	if err != nil || !strings.Contains(pv.User, "## 이 수강생이 인정한 오개념") {
		t.Fatalf("prompt preview shows the block (%v)", err)
	}

	// 같은 목표를 혼자 맞히면(unaided_correct) 고친 오개념이 된다.
	if _, err := s.AbandonCheck(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	pull(t, s, user, f.objA, `next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 second'), last_fail_at = strftime('%Y-%m-%dT%H:%M:%fZ', last_fail_at, '-2 days')`)
	rd, _, err := s.StartReview(ctx, user)
	if err != nil || len(rd.Items) != 1 || rd.Items[0].ObjectiveID != f.objA {
		t.Fatalf("review = %+v (%v)", rd.Items, err)
	}
	if rd = selfGrade(t, s, user, rd.Session.ID, "메모리에서 읽어요", "correct"); lastJudged(rd).Outcome != learning.OutcomeUnaidedCorrect {
		t.Fatalf("review outcome = %+v", lastJudged(rd))
	}
	resolved, err := s.ListMisconceptions(ctx, user, learning.MisconceptionResolved)
	if err != nil || len(resolved) != 1 || resolved[0].ID != m.ID || resolved[0].ResolvedAt == nil {
		t.Fatalf("resolved = %+v (%v)", resolved, err)
	}
	if open, _ := s.ListMisconceptions(ctx, user, learning.MisconceptionOpen); len(open) != 0 {
		t.Fatalf("nothing open after resolution: %+v", open)
	}
	if inst, err = s.instantTarget(ctx, user, f.conceptID, 0, ""); err != nil || len(inst.LectureContext().Misconceptions) != 0 {
		t.Fatalf("a resolved misconception leaves the prompt: %+v (%v)", inst.LectureContext().Misconceptions, err)
	}
}

// TestTodayRecheckCard: a lapsed (≥ 2, not recovered) prerequisite of the goal
// path's next concept gets a "다시 확인" card; recovery (lapses 0) removes it.
func TestTodayRecheckCard(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	db := testDB(s)
	pre := "test.recheck-pre-" + strings.TrimPrefix(user, "test-prompts-")
	obj := extraConcept(t, s, user, pre, "선행 개념", "completed", learning.BestUnaided, 1, f.conceptID)
	if _, err := db.ExecContext(ctx, `INSERT INTO user_goals (id, user_id, title, target_concept_ids) VALUES (?, ?, '목표', json_array(?))`,
		store.NewID(), user, f.conceptID); err != nil {
		t.Fatal(err)
	}

	today, err := s.Today(ctx, user, 5)
	if err != nil || len(today.Recheck) != 0 || len(today.Items) == 0 || today.Items[0].Kind != learning.RecGoalPath {
		t.Fatalf("one lapse → no card: %+v (%v)", today, err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE user_objective_mastery SET lapses = 2, next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 second') WHERE user_id = ? AND objective_id = ?`, user, obj); err != nil {
		t.Fatal(err)
	}
	today, err = s.Today(ctx, user, 5)
	if err != nil || len(today.Recheck) != 1 {
		t.Fatalf("recheck = %+v (%v)", today.Recheck, err)
	}
	card := today.Recheck[0]
	if card.ConceptID != pre || card.ConceptName != "선행 개념" || !card.Due || card.GoalID == "" || today.Exhausted || today.Items[0].Kind != learning.RecGoalPath {
		t.Fatalf("card = %+v exhausted=%v", card, today.Exhausted)
	}
	if _, err := db.ExecContext(ctx, `UPDATE user_objective_mastery SET lapses = 0 WHERE user_id = ? AND objective_id = ?`, user, obj); err != nil {
		t.Fatal(err)
	}
	if today, _ = s.Today(ctx, user, 5); len(today.Recheck) != 0 {
		t.Fatalf("recovered → no card: %+v", today.Recheck)
	}
}
