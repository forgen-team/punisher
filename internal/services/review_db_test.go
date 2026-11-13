package services

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/internal/store"
)

// checkBoth runs the lecture's check with both objectives answered correctly
// (box 1, next review tomorrow) and leaves before the integrative item.
func checkBoth(t *testing.T, s *Service, fake *checkLLM, user string, f checkFixture) models.CheckDetail {
	t.Helper()
	ctx := context.Background()
	fake.push("items", itemSetJSON(f, true))
	d, _, err := s.StartCheck(ctx, user, "check", f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	id := d.Session.ID
	waitIdle(t, s, user, id)
	fake.push("tutor", tutorJSON("correct", "next", "맞아요.", 0.9), tutorJSON("correct", "next", "맞아요.", 0.9))
	answer(t, s, user, id, "메모리에서 읽어요")
	answer(t, s, user, id, "더티 페이지로 남아요")
	if d, err = s.AbandonCheck(ctx, user, id); err != nil {
		t.Fatal(err)
	}
	return d
}

// testDB is the SQLite handle (the witems test uses Pool(); "?" placeholders, no NOW() rewrite).
func testDB(s *Service) *sql.DB { return s.Store.(*store.SQLiteStore).DB() }

// pull moves an objective's mastery back in time ("시각을 당긴 데이터").
func pull(t *testing.T, s *Service, user, objectiveID, set string) {
	t.Helper()
	if _, err := testDB(s).ExecContext(context.Background(), `UPDATE user_objective_mastery SET `+set+`
		WHERE user_id = ? AND objective_id = ?`, user, objectiveID); err != nil {
		t.Fatal(err)
	}
}

// dueNow makes an objective due an hour ago with its last credit / failure two days back.
func dueNow(t *testing.T, s *Service, user, objectiveID string) {
	pull(t, s, user, objectiveID, `next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour'),
		last_credit_at = strftime('%Y-%m-%dT%H:%M:%fZ', last_credit_at, '-2 days'), last_fail_at = strftime('%Y-%m-%dT%H:%M:%fZ', last_fail_at, '-2 days')`)
}

func selfGrade(t *testing.T, s *Service, user, id, text, verdict string) models.CheckDetail {
	t.Helper()
	ctx := context.Background()
	if _, err := s.AnswerCheck(ctx, user, id, text); err != nil {
		t.Fatal(err)
	}
	d, err := s.SelfGradeCheck(ctx, user, id, verdict)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func askTurn(d models.CheckDetail, itemID string) models.CheckTurn {
	for _, tr := range d.Turns {
		if tr.ItemID == itemID && tr.Move == "ask" {
			return tr
		}
	}
	return models.CheckTurn{}
}

// TestReviewSelfAndTutor: queue → session (self) → boxes and dates change →
// nothing left → tutored review narrows once on a reused item → metrics.
func TestReviewSelfAndTutor(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	if sum, err := s.ReviewSummary(ctx, user); err != nil || sum.Due != 0 || sum.Today != nil {
		t.Fatalf("nothing due right after the check: %+v %v", sum, err)
	}
	if _, _, err := s.StartReview(ctx, user); !errors.Is(err, ErrNoReviews) {
		t.Fatalf("no due → ErrNoReviews, got %v", err)
	}
	dueNow(t, s, user, f.objA)
	dueNow(t, s, user, f.objB)
	sum, err := s.ReviewSummary(ctx, user)
	if err != nil || sum.Due != 2 || sum.Overdue != 0 || sum.EstimatedMinutes != 2 || sum.Grader != "self" || sum.Throughput != 10 {
		t.Fatalf("summary = %+v (%v)", sum, err)
	}

	d, created, err := s.StartReview(ctx, user)
	if err != nil || !created || d.Session.Kind != "review" || d.Session.Grader != "self" || len(d.Items) != 2 {
		t.Fatalf("review session = %+v created=%v (%v)", d.Session, created, err)
	}
	id := d.Session.ID
	var key string
	if err := testDB(s).QueryRowContext(ctx, `SELECT session_key FROM check_sessions WHERE id = ?`, id).Scan(&key); err != nil ||
		key != ReviewSessionKey(time.Now()) {
		t.Fatalf("session key = %q (%v)", key, err)
	}
	if again, created, err := s.StartReview(ctx, user); err != nil || created || again.Session.ID != id {
		t.Fatalf("POST again returns the open session: %+v %v %v", again.Session, created, err)
	}
	for _, it := range d.Items {
		var purpose string
		if err := testDB(s).QueryRowContext(ctx, `SELECT purpose FROM check_items WHERE id = ?`, it.ID).Scan(&purpose); err != nil || purpose != "review" {
			t.Fatalf("an unseen item (the review variant) comes first: %s %v", purpose, err)
		}
	}
	if askTurn(d, d.Items[0].ID).ReusedItem {
		t.Fatal("unseen item must not be marked reused")
	}
	first := d.Items[0].ObjectiveID
	d = selfGrade(t, s, user, id, "메모리에 남아요", "correct")
	if m := masteryOf(t, s, user, f.conceptID, first); m.Box != 2 || m.Level != learning.LevelRemember || m.NextReviewAt == nil ||
		!m.NextReviewAt.Equal(learning.KSTDayStart(time.Now()).AddDate(0, 0, 7)) {
		t.Fatalf("self correct → box 2, +7 days: %+v", m)
	}
	second := d.Items[1].ObjectiveID
	d = selfGrade(t, s, user, id, "모르겠어요", "incorrect")
	if d.Session.Status != "finished" || lastJudged(d).Outcome != learning.OutcomeFailed {
		t.Fatalf("finished = %+v %+v", d.Session, lastJudged(d))
	}
	if m := masteryOf(t, s, user, f.conceptID, second); m.Box != 0 || m.Best != learning.BestUnaided || m.NextReviewAt == nil {
		t.Fatalf("self incorrect → box 0, best kept: %+v", m)
	}
	if sum, _ = s.ReviewSummary(ctx, user); sum.Due != 0 || sum.Today == nil || sum.Today.ID != id || sum.Today.Done != 2 || sum.Today.Total != 2 {
		t.Fatalf("summary after = %+v today=%+v", sum, sum.Today)
	}

	// 튜터 채점: 설정은 다음 세션부터. 모든 문항을 봤으므로 재사용(reused_item) 표시.
	grader := learning.GraderLLM
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{ReviewGrader: &grader}); err != nil {
		t.Fatal(err)
	}
	dueNow(t, s, user, second)
	d, created, err = s.StartReview(ctx, user)
	if err != nil || !created || d.Session.Grader != "llm" || len(d.Items) != 1 || !askTurn(d, d.Items[0].ID).ReusedItem {
		t.Fatalf("tutored review = %+v items=%d (%v)", d.Session, len(d.Items), err)
	}
	tid := d.Session.ID
	fake.push("tutor", tutorJSON("partial", "narrow", "디스크에는 언제 쓰일까요?", 0.9),
		tutorJSON("incorrect", "wrap", "fsync 가 기록을 기다립니다.", 0.9))
	d = answer(t, s, user, tid, "메모리에 남아요")
	if lastTurn(d).Move != "narrow" || !d.Current.CanAnswer {
		t.Fatalf("narrow = %+v %+v", lastTurn(d), d.Current)
	}
	d = answer(t, s, user, tid, "바로 써져요")
	if !strings.HasSuffix(fake.users["tutor"][len(fake.users["tutor"])-1], "wrap\n") {
		t.Fatalf("after one narrowing only the closing move is allowed: %q", fake.users["tutor"][len(fake.users["tutor"])-1])
	}
	j := lastJudged(d)
	if d.Session.Status != "finished" || j.Outcome != learning.OutcomeFailed || !j.ReusedItem || j.Grader != "llm" {
		t.Fatalf("tutored review end = %+v %+v", d.Session, j)
	}

	m, err := s.Metrics(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range m.Buckets {
		switch {
		case b.IntervalDays == 1 && b.Grader == "self":
			if b.Attempts != 2 || b.Correct != 1 || b.Rate == nil || *b.Rate != 0.5 {
				t.Fatalf("1-day self bucket = %+v", b)
			}
		case b.Attempts != 0 || b.Rate != nil:
			t.Fatalf("reused / other buckets stay empty: %+v", b)
		}
	}

	hist, next, err := s.Store.ListCheckHistory(ctx, user, "", 2, "")
	if err != nil || len(hist) != 2 || next == "" || hist[0].ID != tid || hist[0].ReviewDay == "" || hist[1].ID != id || hist[1].UnaidedCorrect != 1 {
		t.Fatalf("history page 1 = %+v next=%q (%v)", hist, next, err)
	}
	rest, next, err := s.Store.ListCheckHistory(ctx, user, "", 2, next)
	if err != nil || len(rest) != 1 || next != "" || rest[0].Kind != "check" || rest[0].LectureTitle == "" {
		t.Fatalf("history page 2 = %+v next=%q (%v)", rest, next, err)
	}
	if only, _, _ := s.Store.ListCheckHistory(ctx, user, "check", 10, ""); len(only) != 1 {
		t.Fatalf("kind filter = %+v", only)
	}
}

// TestReviewDayRollover: yesterday's open review session is abandoned by the
// next POST (and by the sweep); its applied evidence stays.
func TestReviewDayRollover(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	dueNow(t, s, user, f.objA)
	dueNow(t, s, user, f.objB)
	d, _, err := s.StartReview(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	old := d.Session.ID
	first := d.Items[0].ObjectiveID
	selfGrade(t, s, user, old, "메모리", "correct")
	if _, err := testDB(s).ExecContext(ctx, `UPDATE check_sessions SET session_key = 'review:2000-01-01' WHERE id = ?`, old); err != nil {
		t.Fatal(err)
	}
	d, created, err := s.StartReview(ctx, user)
	if err != nil || !created || d.Session.ID == old || len(d.Items) != 1 {
		t.Fatalf("new day's session = %+v created=%v (%v)", d.Session, created, err)
	}
	if got, _ := s.Store.GetCheckSession(ctx, user, old); got.Status != "abandoned" || got.FinishedAt == nil {
		t.Fatalf("yesterday's session = %+v", got)
	}
	if m := masteryOf(t, s, user, f.conceptID, first); m.Box != 2 {
		t.Fatalf("evidence of the abandoned session stays: %+v", m)
	}
	if _, err := testDB(s).ExecContext(ctx, `UPDATE check_sessions SET session_key = 'review:2000-01-02' WHERE id = ?`, d.Session.ID); err != nil {
		t.Fatal(err)
	}
	if n, err := s.Store.AbandonStaleReviews(ctx, "", ReviewSessionKey(time.Now())); err != nil || n < 1 {
		t.Fatalf("sweep = %d %v", n, err)
	}
	if got, _ := s.Store.GetCheckSession(ctx, user, d.Session.ID); got.Status != "abandoned" {
		t.Fatalf("swept session = %+v", got)
	}
}

// TestReviewBacklogQueuesNewChecks: over three days of throughput a new
// check's objectives wait (queued, result says so); under a day they are
// released oldest first with a review date.
func TestReviewBacklogQueuesNewChecks(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	pool := testDB(s)
	one := 1
	if _, err := s.Store.UpdateLearningSettings(ctx, user, store.LearningSettingsPatch{ReviewMinutes: &one}); err != nil {
		t.Fatal(err)
	}
	for i := 3; i <= 6; i++ { // 처리량 1/일 → 밀린 4개는 3일치 초과
		obj := f.conceptID + "#" + string(rune('0'+i))
		if _, err := pool.ExecContext(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES (?, ?, '밀린 목표', 1, ?)`, obj, f.conceptID, i); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.ExecContext(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at)
			VALUES (?, ?, ?, 1, 'unaided', strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-3 days'))`, user, obj, f.conceptID); err != nil {
			t.Fatal(err)
		}
	}
	d := checkBoth(t, s, fake, user, f)
	var result struct {
		Items []store.ResultEntry `json:"items"`
	}
	if err := json.Unmarshal(d.Session.Result, &result); err != nil || len(result.Items) != 2 || !result.Items[0].Queued || result.Items[0].NextReviewAt != nil {
		t.Fatalf("result must say the objectives wait: %s (%v)", d.Session.Result, err)
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); !a.Queued || a.NextReviewAt != nil || a.Box != 1 {
		t.Fatalf("queued objective = %+v", a)
	}
	sum, err := s.ReviewSummary(ctx, user)
	if err != nil || sum.Due != 4 || sum.Queued != 2 || !sum.Backlogged || sum.Overdue != 4 {
		t.Fatalf("backlogged summary = %+v (%v)", sum, err)
	}
	if today, err := s.Today(ctx, user, 5); err != nil || today.Review.Due != 4 || !today.Review.Backlogged || today.Review.Queued != 2 ||
		today.Review.EstimatedMinutes != 1 || today.Review.LectureMinutes != 30 {
		t.Fatalf("today review = %+v (%v)", today.Review, err)
	}

	// 밀린 복습이 1일치(1개) 밑으로: 오래된 대기 1개만 큐로.
	if _, err := pool.ExecContext(ctx, `UPDATE user_objective_mastery SET next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+5 days')
		WHERE user_id = ? AND objective_id LIKE ?`, user, f.conceptID+"#_"); err != nil {
		t.Fatal(err)
	}
	pull(t, s, user, f.objB, `updated_at = strftime('%Y-%m-%dT%H:%M:%fZ', updated_at, '+1 minute')`)
	if sum, err = s.ReviewSummary(ctx, user); err != nil || sum.Queued != 1 {
		t.Fatalf("one released: %+v (%v)", sum, err)
	}
	a, b := masteryOf(t, s, user, f.conceptID, f.objA), masteryOf(t, s, user, f.conceptID, f.objB)
	if a.Queued || a.NextReviewAt == nil || !b.Queued {
		t.Fatalf("the older waiting objective is released first: A=%+v B=%+v", a, b)
	}
	if a.NextReviewAt.Before(time.Now().Add(learning.MinCreditGap - time.Minute)) {
		t.Fatalf("released date = %v", a.NextReviewAt)
	}
}

// TestReviewGraduation: box 3 correct → 90-day graduation review → correct → done (out of the queue, level kept).
func TestReviewGraduation(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	pull(t, s, user, f.objA, `box = 3, last_credit_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-40 days'), next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour')`)
	d, _, err := s.StartReview(ctx, user)
	if err != nil || len(d.Items) != 1 {
		t.Fatalf("review = %+v (%v)", d.Items, err)
	}
	selfGrade(t, s, user, d.Session.ID, "메모리", "correct")
	a := masteryOf(t, s, user, f.conceptID, f.objA)
	if a.GradStep != learning.GradPending || a.NextReviewAt == nil || a.NextReviewAt.Before(time.Now().AddDate(0, 0, 89)) {
		t.Fatalf("graduation review scheduled at +90 days: %+v", a)
	}
	pull(t, s, user, f.objA, `last_credit_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-100 days'), next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour')`)
	d, _, err = s.StartReview(ctx, user)
	if err != nil || len(d.Items) != 1 || !askTurn(d, d.Items[0].ID).ReusedItem {
		t.Fatalf("graduation review (all items seen → reused) = %+v (%v)", d.Turns, err)
	}
	selfGrade(t, s, user, d.Session.ID, "메모리", "correct")
	a = masteryOf(t, s, user, f.conceptID, f.objA)
	if a.GradStep != learning.GradDone || a.NextReviewAt != nil || a.Level != learning.LevelRemember {
		t.Fatalf("graduated = %+v", a)
	}
	pull(t, s, user, f.objA, `next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour')`)
	if q, err := s.Store.ReviewQueue(ctx, user, time.Now()); err != nil || len(q) != 0 {
		t.Fatalf("graduated objectives leave the queue: %+v (%v)", q, err)
	}
}

// TestReviewVariantPregeneration: a soon-due objective whose items were all
// seen gets one review variant (check budget, idempotent per day); the next
// session picks it as unseen.
func TestReviewVariantPregeneration(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	checkBoth(t, s, fake, user, f)
	dueNow(t, s, user, f.objA)
	d, _, err := s.StartReview(ctx, user)
	if err != nil {
		t.Fatal(err)
	}
	selfGrade(t, s, user, d.Session.ID, "메모리", "correct") // A 의 check·review 문항을 모두 봄
	pull(t, s, user, f.objA, `next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '+20 hours')`)
	s.Cfg.OwnerUserID = user

	s.Cfg.DailyCheckTokenLimit = 1
	if _, err := s.PregenerateReviewVariants(ctx); !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("over the check budget nothing is generated: %v", err)
	}
	s.Cfg.DailyCheckTokenLimit = 0
	fake.push("items", `{"items":[`+itemJSON(f.objA, "review", "apply", "캐시가 꽉 차면 다시 읽기는 어떻게 될까요?", "s1")+`]}`)
	rep, err := s.PregenerateReviewVariants(ctx)
	if err != nil || rep.Generated != 1 {
		t.Fatalf("variants = %+v (%v)", rep, err)
	}
	if !strings.Contains(fake.users["items"][len(fake.users["items"])-1], "복습 변형") {
		t.Fatal("the variant prompt asks for review variants")
	}
	var variantOf, purpose string
	if err := testDB(s).QueryRowContext(ctx, `SELECT COALESCE(variant_of, ''), purpose FROM check_items
		WHERE concept_id = ? AND prompt LIKE '캐시가 꽉%'`, f.conceptID).Scan(&variantOf, &purpose); err != nil || variantOf == "" || purpose != "review" {
		t.Fatalf("variant row = %q %q (%v)", variantOf, purpose, err)
	}
	if rep, err = s.PregenerateReviewVariants(ctx); err != nil || rep.Generated != 0 {
		t.Fatalf("an unseen item exists now: %+v (%v)", rep, err)
	}
	pull(t, s, user, f.objA, `next_review_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 hour'), last_credit_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-2 days')`)
	d, _, err = s.StartReview(ctx, user)
	if err != nil || len(d.Items) != 1 || !strings.HasPrefix(d.Items[0].Prompt, "캐시가 꽉") || askTurn(d, d.Items[0].ID).ReusedItem {
		t.Fatalf("the new variant is picked unseen: %+v (%v)", d.Items, err)
	}
}
