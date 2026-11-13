package services

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// startedCheck opens a check session of the fixture lecture (items A, B, integrative) and waits for its first ask.
func startedCheck(t *testing.T, s *Service, fake *checkLLM, user string, f checkFixture) string {
	t.Helper()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(context.Background(), user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	waitIdle(t, s, user, d.Session.ID)
	return d.Session.ID
}

// waitTurnDone polls until the turn is no longer pending (a held gate keeps it pending).
func waitTurnDone(t *testing.T, s *Service, user, id, turnID string) models.CheckTurn {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		turns, err := s.Store.ListCheckTurns(context.Background(), user, id)
		if err != nil {
			t.Fatal(err)
		}
		for _, tr := range turns {
			if tr.ID == turnID && tr.Status != "pending" {
				return tr
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("turn %s still pending", turnID)
	return models.CheckTurn{}
}

// TestRevealWhileGradingIsBusy (MAJOR 1): '그냥 알려줘' sent while the answer is
// being graded is 409 busy (checked inside the item transaction), and the
// grade then lands on its own item — no panic in the background run.
func TestRevealWhileGradingIsBusy(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	id := startedCheck(t, s, fake, user, f)
	fake.hold = make(chan struct{})
	fake.push("tutor", tutorJSON("correct", "next", "맞아요, 메모리에서 읽습니다.", 0.9))
	d, err := s.AnswerCheck(ctx, user, id, "메모리에서 읽어서요")
	if err != nil {
		t.Fatal(err)
	}
	pending := lastTurn(d)
	if pending.Status != "pending" {
		t.Fatalf("the grade must be running: %+v", pending)
	}
	if _, err := s.RevealCheck(ctx, user, id); !errors.Is(err, ErrCheckBusy) {
		t.Fatalf("reveal during grading = %v, want ErrCheckBusy", err)
	}
	// 서비스 검사를 지나 저장소까지 온 reveal(경쟁)도 트랜잭션 안에서 막힌다.
	st, err := s.loadCheckState(ctx, user, id)
	if err != nil {
		t.Fatal(err)
	}
	reveal := store.TurnWrite{ItemID: st.items[0].ID, Role: "tutor", Move: check.MoveReveal, Verdict: learning.VerdictRevealed, Grader: learning.GraderLLM}
	if err := s.finishItem(ctx, user, st, st.items[0], learning.ItemResult{Revealed: true}, false, reveal, false); !errors.Is(err, store.ErrCheckBusy) {
		t.Fatalf("store-level reveal under a pending turn = %v, want store.ErrCheckBusy", err)
	}
	close(fake.hold)
	d = waitIdle(t, s, user, id)
	if d.Session.Cursor != 1 || d.Items[0].Outcome != learning.OutcomeUnaidedCorrect {
		t.Fatalf("the grade must close item 1 normally: cursor=%d item=%+v", d.Session.Cursor, d.Items[0])
	}
}

// TestPendingTurnOnClosedItemFails (MAJOR 1): a pending grade whose item is no
// longer under the cursor fails instead of grading the new current item (that
// item has no answer: index -1 panic before), and an answer for a closed item
// is refused by StartCheckTurns.
func TestPendingTurnOnClosedItemFails(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	id := startedCheck(t, s, fake, user, f)
	fake.hold = make(chan struct{})
	fake.push("tutor", tutorJSON("correct", "next", "맞아요.", 0.9))
	d, err := s.AnswerCheck(ctx, user, id, "메모리에서 읽어서요")
	if err != nil {
		t.Fatal(err)
	}
	pending := lastTurn(d)
	// 예전 경쟁의 결과: 채점 중에 문항이 닫혀 커서가 다음 문항으로 넘어간 상태.
	if _, err := testDB(s).ExecContext(ctx, `UPDATE check_sessions SET cursor = 1 WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	close(fake.hold)
	if got := waitTurnDone(t, s, user, id, pending.ID); got.Status != "failed" || got.Verdict != "" {
		t.Fatalf("a grade of a closed item must fail: %+v", got)
	}
	d, err = s.GetCheck(ctx, user, id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Items[1].Done || d.Session.Status != "active" {
		t.Fatalf("the new current item must stay untouched: %+v", d.Items[1])
	}
	// 문항이 이미 넘어간 뒤에 실행이 시작돼도(상태를 다시 읽음) 새 현재 문항을 채점하지 않는다.
	orphan := store.NewID()
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO check_turns (id, session_id, user_id, item_id, seq, role, move, text, status, grader)
		SELECT ?1, ?2, ?3, ?4, MAX(seq) + 1, 'tutor', '', '', 'pending', 'llm' FROM check_turns WHERE session_id = ?2`,
		orphan, id, user, d.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := s.runCheckTurn(ctx, "key", user, id, orphan); err != nil {
		t.Fatal(err)
	}
	if got := waitTurnDone(t, s, user, id, orphan); got.Status != "failed" || fake.count("tutor") != 1 {
		t.Fatalf("a run for a closed item must fail without calling the tutor: %+v calls=%d", got, fake.count("tutor"))
	}
	learner := &store.TurnWrite{ItemID: d.Items[0].ID, Role: "learner", Move: check.MoveAnswer, Text: "늦은 답"}
	tutor := &store.TurnWrite{ItemID: d.Items[0].ID, Role: "tutor", Status: "pending", Grader: learning.GraderLLM}
	if _, err := s.Store.StartCheckTurns(ctx, user, id, learner, tutor, maxPendingChecks, true); !errors.Is(err, store.ErrCheckClosed) {
		t.Fatalf("an answer for an item not under the cursor = %v, want ErrCheckClosed", err)
	}
}

// TestRunDetachedRecoversPanic (MAJOR 1): a panic in a background run is
// recovered and handed to fail (the turn/job is recorded failed), the process lives on.
func TestRunDetachedRecoversPanic(t *testing.T) {
	failed := make(chan error, 1)
	runDetached("test", "x", time.Second, func(context.Context) error {
		var answers []models.CheckTurn
		_ = answers[len(answers)-1] // the old tutorReply panic
		return nil
	}, func(_ context.Context, err error) { failed <- err })
	select {
	case err := <-failed:
		if !strings.Contains(err.Error(), "internal error") {
			t.Fatalf("fail got %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the panic must reach fail")
	}
}

// failItemA answers item A of a fresh check session to a failure (partial →
// narrow → incorrect → explain → incorrect) and returns the session and the judgement.
func failItemA(t *testing.T, s *Service, fake *checkLLM, user string, f checkFixture) (string, models.CheckTurn) {
	t.Helper()
	id := startedCheck(t, s, fake, user, f)
	fake.push("tutor", tutorJSON("partial", "narrow", "디스크에는 언제 쓰일까요?", 0.9),
		tutorJSON("incorrect", "explain", "읽은 블록은 메모리에 남습니다. 그러면 두 번째 읽기는 어디서 할까요?", 0.9),
		tutorJSON("incorrect", "next", "다음 문항으로 가요.", 0.9))
	answer(t, s, user, id, "메모리에 남아요")
	answer(t, s, user, id, "디스크요")
	d := answer(t, s, user, id, "모르겠어요")
	judged := lastJudged(d)
	if judged.Outcome != learning.OutcomeFailed || d.Session.Cursor != 1 {
		t.Fatalf("item A = %+v cursor=%d", judged, d.Session.Cursor)
	}
	return id, judged
}

func disputeA(t *testing.T, s *Service, fake *checkLLM, user, id, judgedID string) models.CheckTurn {
	t.Helper()
	fake.push("regrade", `{"verdict":"correct","met":["메모리"],"missing":[],"quote":"메모리에 남아요","confidence":0.8}`)
	if _, err := s.DisputeCheck(context.Background(), user, id, judgedID, ""); err != nil {
		t.Fatal(err)
	}
	disp := lastTurn(waitIdle(t, s, user, id))
	if disp.Move != check.MoveDispute || disp.Grade == nil || disp.Grade.Accepted == nil || !*disp.Grade.Accepted {
		t.Fatalf("dispute = %+v", disp)
	}
	return disp
}

// TestDisputeAfterIntegrativeCreditRestores (MAJOR 2): the session ends with a
// defended integrative item (credit on every objective) and item A is disputed
// on the result screen — restored, not a conflict, the credit kept.
func TestDisputeAfterIntegrativeCreditRestores(t *testing.T) {
	s, fake, user, f := checkService(t)
	id, judged := failItemA(t, s, fake, user, f)
	fake.push("tutor", tutorJSON("correct", "next", "맞아요.", 0.9),
		tutorJSON("correct", "challenge", "체크포인트 없이 WAL 만으로 충분하지 않나요?", 0.9),
		tutorJSON("correct", "wrap", "반론을 잘 막았습니다.", 0.9))
	answer(t, s, user, id, "더티 페이지로 남아요")
	answer(t, s, user, id, "읽기는 캐시, 쓰기는 더티 페이지라 체크포인트가 필요해요")
	if d := answer(t, s, user, id, "WAL 이 길어지면 복구가 느려져요"); d.Session.Status != "finished" {
		t.Fatalf("session = %+v", d.Session)
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); !a.IntegrativeOK || a.Box != 0 {
		t.Fatalf("A before the dispute = %+v", a)
	}
	disp := disputeA(t, s, fake, user, id, judged.ID)
	if disp.Grade.Conflict || !strings.Contains(disp.Text, "되돌렸어요") {
		t.Fatalf("an integrative credit is no later evidence: %+v", disp)
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	if a.Best != learning.BestUnaided || !a.IntegrativeOK || a.Box != 0 || a.NextReviewAt != nil || a.LastGrader != learning.GraderRegrade {
		t.Fatalf("A after the dispute = %+v (restored to before the item, credit kept)", a)
	}
}

// TestDisputeAfterLaterEvidenceConflicts (MAJOR 2): evidence that reached the
// objective after the judged item (the marker moved) keeps the box, and the
// tutor says so instead of "되돌렸어요".
func TestDisputeAfterLaterEvidenceConflicts(t *testing.T) {
	s, fake, user, f := checkService(t)
	id, judged := failItemA(t, s, fake, user, f)
	// 그 사이 다른 근거(다른 세션의 판정)가 반영된 상태: 표지가 다른 턴을 가리키고 상자가 올라가 있다.
	pull(t, s, user, f.objA, `box = 2, last_evidence_turn_id = '`+store.NewID()+`'`)
	disp := disputeA(t, s, fake, user, id, judged.ID)
	if !disp.Grade.Conflict || !strings.Contains(disp.Text, "상자는 그대로") || strings.Contains(disp.Text, "되돌렸어요") {
		t.Fatalf("later evidence must be a conflict with its own message: %+v", disp)
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); a.Box != 2 || a.Best != learning.BestUnaided {
		t.Fatalf("conflict keeps the box, raises best: %+v", a)
	}
}

// TestDisputeAfterReleaseKeepsRelease (MAJOR 2): an objective that was waiting
// in the inflow queue before the item, released afterwards, does not wait
// again when the dispute restores the before-state.
func TestDisputeAfterReleaseKeepsRelease(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	if _, err := testDB(s).ExecContext(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best,
		objective_text, objective_depth, queued) SELECT ?, o.id, o.concept_id, 1, 'aided', o.text, o.depth, 1 FROM objectives o WHERE o.id = ?`,
		user, f.objA); err != nil {
		t.Fatal(err)
	}
	id, judged := failItemA(t, s, fake, user, f)
	if a := masteryOf(t, s, user, f.conceptID, f.objA); !a.Queued || a.NextReviewAt != nil {
		t.Fatalf("a waiting objective keeps waiting after the item: %+v", a)
	}
	// 대기 해제(ReleaseQueuedReviews 와 같은 갱신 — 근거 표지는 건드리지 않는다).
	pull(t, s, user, f.objA, `queued = 0, next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+1 day')`)
	disp := disputeA(t, s, fake, user, id, judged.ID)
	if disp.Grade.Conflict {
		t.Fatalf("a release is no later evidence: %+v", disp)
	}
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	if a.Queued || a.NextReviewAt == nil || a.Box != 1 || a.Best != learning.BestUnaided {
		t.Fatalf("restored A must stay released with a review date: %+v", a)
	}
}

// TestSelfGradeOneAnswerAndNoDispute (m3, m6): a self-graded item takes one
// answer even when a second submit passes the service check, and a self grade
// cannot be disputed.
func TestSelfGradeOneAnswerAndNoDispute(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	dueNow(t, s, user, f.objA)
	d, _, err := s.StartReview(ctx, user)
	if err != nil || d.Session.Grader != learning.GraderSelf {
		t.Fatalf("self review = %+v (%v)", d.Session, err)
	}
	id := d.Session.ID
	if _, err := s.AnswerCheck(ctx, user, id, "메모리요"); err != nil {
		t.Fatal(err)
	}
	second := &store.TurnWrite{ItemID: d.Items[0].ID, Role: "learner", Move: check.MoveAnswer, Text: "두 번째"}
	if _, err := s.Store.StartCheckTurns(ctx, user, id, second, nil, maxPendingChecks, true); !errors.Is(err, store.ErrCheckClosed) {
		t.Fatalf("a second self-graded answer = %v, want ErrCheckClosed", err)
	}
	d, err = s.SelfGradeCheck(ctx, user, id, learning.VerdictIncorrect)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.DisputeCheck(ctx, user, id, lastJudged(d).ID, ""); !errors.Is(err, ErrCheckState) {
		t.Fatalf("dispute of a self grade = %v, want ErrCheckState (409)", err)
	}
}

// TestBadIDsAreNotFound (MAJOR 3): ids are compared as uuid; a malformed id is 404, not a cast error.
func TestBadIDsAreNotFound(t *testing.T) {
	s, _, user, _ := checkService(t)
	ctx := context.Background()
	if _, err := s.GetCheck(ctx, user, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad session id = %v", err)
	}
	if _, _, err := s.AcknowledgeMisconception(ctx, user, "x'; --"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad turn id = %v", err)
	}
	if _, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposePlacement, ChapterID: "chapter-1"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad chapter id = %v", err)
	}
}

// TestDiagnosticCountsTowardPendingLimit (MEDIUM): a diagnostic generation has
// no job row, so its preparing session counts toward the user's limit — a new
// verification is 409 busy over it (a pretest is skipped), an open one is still returned.
func TestDiagnosticCountsTowardPendingLimit(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	fake.hold = make(chan struct{}) // 진단 문항 생성(check.pretest 블록)도 막혀 세션이 preparing 으로 남는다.
	defer close(fake.hold)
	open, created, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposeKnownVerify, ConceptID: f.conceptID})
	if err != nil || !created || open == nil {
		t.Fatalf("first verification = %v %v", created, err)
	}
	for i := 0; i < maxPendingChecks; i++ {
		if _, err := testDB(s).ExecContext(ctx, `INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, status)
			VALUES (?, ?, 'check_items', ?, 'running')`, store.NewID(), "test:"+store.RandomToken(), user); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := s.Store.CountPendingChecks(ctx, user); err != nil || n < maxPendingChecks+1 {
		t.Fatalf("pending = %d (%v): the preparing verification must count", n, err)
	}
	again, created, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposeKnownVerify, ConceptID: f.conceptID})
	if err != nil || created || again.Session.ID != open.Session.ID {
		t.Fatalf("the open verification must be returned over the limit: %v %v", created, err)
	}
	if _, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposePlacement, ChapterID: "00000000-0000-0000-0000-000000000000"}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("unknown chapter = %v", err)
	}
	if _, err := testDB(s).ExecContext(ctx, `UPDATE check_sessions SET status = 'failed' WHERE id = ?`, open.Session.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.RetryCheckSession(ctx, user, open.Session.ID, maxPendingChecks); !errors.Is(err, store.ErrCheckBusy) {
		t.Fatalf("retry over the limit = %v, want ErrCheckBusy", err)
	}
	if _, _, err := s.StartDiagnostic(ctx, user, DiagnosticRequest{Kind: check.PurposeKnownVerify, ConceptID: f.conceptID}); !errors.Is(err, ErrCheckBusy) {
		t.Fatalf("a new verification over the limit = %v, want ErrCheckBusy", err)
	}
}
