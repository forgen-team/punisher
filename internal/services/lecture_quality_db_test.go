package services

import (
	"context"
	"errors"
	"github.com/forgen-team/punisher/internal/i18n"
	"strings"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// TestCheckNeedsPassedReview (v2.2 §2.1): a lecture without a pass review gets
// no check items — ErrLectureNotReviewed (409 LECTURE_NOT_REVIEWED), no
// session, no LLM call — and moves to the front of the backfill queue.
func TestCheckNeedsPassedReview(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	db := testDB(s)
	if _, err := db.ExecContext(ctx, `DELETE FROM lecture_reviews WHERE lecture_id = ?`, f.lectureID); err != nil {
		t.Fatal(err)
	}
	// 기존 강의(minutes_estimated 없음)는 조회 때 본문으로 시간을 계산한다(짧은 섹션 2개 → 섹션마다 최소 1분, 합 2분).
	if lec, err := s.Store.GetLecture(ctx, user, f.lectureID); err != nil || lec.ReviewState != models.ReviewUnreviewed || lec.EstimatedMinutes != 2 {
		t.Fatalf("review state = %q minutes = %d (%v)", lec.ReviewState, lec.EstimatedMinutes, err)
	}
	if _, _, err := s.StartCheck(ctx, user, "check", f.lectureID); !errors.Is(err, ErrLectureNotReviewed) {
		t.Fatalf("err = %v, want ErrLectureNotReviewed", err)
	}
	var sessions int
	var requested bool
	_ = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM check_sessions WHERE user_id = ?`, user).Scan(&sessions)
	_ = db.QueryRowContext(ctx, `SELECT review_requested_at IS NOT NULL FROM lectures WHERE id = ?`, f.lectureID).Scan(&requested)
	if sessions != 0 || fake.count("items") != 0 || !requested {
		t.Fatalf("sessions=%d item calls=%d requested=%v", sessions, fake.count("items"), requested)
	}
	queue, err := s.Store.LecturesToReview(ctx, 1)
	if err != nil || len(queue) != 1 || queue[0].ID != f.lectureID {
		t.Fatalf("the requested lecture leads the backfill queue: %+v (%v)", queue, err)
	}
}

// TestBackfillReviewPassAndFlag (v2.2 §2.2): requested lectures are reviewed
// first, at most BackfillReviewsPerTick; a pass is recorded and clears the
// flag, a fail flags the lecture with short reasons; neither is reviewed again.
func TestBackfillReviewPassAndFlag(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	db := testDB(s)
	second := store.NewID()
	if _, err := db.ExecContext(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title, content, requested_by)
		SELECT ?, concept_id, objective_ids, objective_key, depth, kind, status, 2, title, content, requested_by FROM lectures WHERE id = ?`,
		second, f.lectureID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE FROM lecture_reviews WHERE lecture_id = ?`, f.lectureID); err != nil {
		t.Fatal(err)
	}
	// 앞선 실패 표시(예전 즉석 강의의 flagged)는 통과하면 풀린다.
	if _, err := db.ExecContext(ctx, `UPDATE lectures SET flagged = 1, flag_reason = 'old',
		review_requested_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-2 minutes') WHERE id = ?`, f.lectureID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `UPDATE lectures SET review_requested_at = strftime('%Y-%m-%dT%H:%M:%fZ', 'now', '-1 minute') WHERE id = ?`, second); err != nil {
		t.Fatal(err)
	}

	s.Cfg.OwnerUserID, s.Cfg.BackfillReviewEnabled = user, false
	if rep, err := s.BackfillReviews(ctx); err != nil || fake.count("tutor") != 0 || len(rep.Passed)+len(rep.Flagged) != 0 {
		t.Fatalf("the switch turns it off: %+v %v", rep, err)
	}
	s.Cfg.BackfillReviewEnabled = true
	fake.push("tutor", reviewPass, reviewMajor) // the reviewer system prompt is not an item/regrade prompt
	rep, err := s.BackfillReviews(ctx)
	if err != nil || strings.Join(rep.Passed, ",") != f.lectureID || strings.Join(rep.Flagged, ",") != second {
		t.Fatalf("report = %+v (%v)", rep, err)
	}
	if u := fake.users["tutor"]; len(u) != 2 || !strings.Contains(u[0], "구조 블록 도입 전 강의") || !strings.Contains(u[0], "## 사실 정확성 규칙") {
		t.Fatal("a stored lecture without material blocks is reviewed with the legacy note and the factual rules")
	}
	passed, _ := s.Store.GetLecture(ctx, user, f.lectureID)
	flagged, _ := s.Store.GetLecture(ctx, user, second)
	if passed.ReviewState != models.ReviewPassed || passed.Flagged || passed.FlagReason != "" || !passed.Reviewed {
		t.Fatalf("passed = %s flagged=%v reason=%q", passed.ReviewState, passed.Flagged, passed.FlagReason)
	}
	if flagged.ReviewState != models.ReviewFlagged || !flagged.Flagged || flagged.FlagReason != i18n.ReasonKo(i18n.ReviewReasonCode("factual_error")) {
		t.Fatalf("flagged = %s reason=%q", flagged.ReviewState, flagged.FlagReason)
	}
	if last := passed.Reviews[len(passed.Reviews)-1]; last.Reviewer != ReviewerBackfill || last.Verdict != "pass" {
		t.Fatalf("backfill review row = %+v", last)
	}
	for _, l := range mustQueue(t, s) {
		if l.ID == f.lectureID || l.ID == second {
			t.Fatalf("a backfilled lecture is not reviewed again: %s", l.ID)
		}
	}
	// 통과한 강의는 확인 문항을 만든다.
	fake.push("items", itemSetJSON(f, true))
	if _, _, err := s.StartCheck(ctx, user, "check", f.lectureID); err != nil {
		t.Fatalf("a passed lecture can be checked: %v", err)
	}
}

func mustQueue(t *testing.T, s *Service) []models.Lecture {
	t.Helper()
	list, err := s.Store.LecturesToReview(context.Background(), 50)
	if err != nil {
		t.Fatal(err)
	}
	return list
}

// TestDisputeAfterRelearnConflicts: '다시 배우기' moves the evidence marker, so a
// dispute of a grade from before it is accepted as a conflict and does not
// overwrite the reset (box stays 0, the conflict text is shown).
func TestDisputeAfterRelearnConflicts(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	id, judged := failItemA(t, s, fake, user, f)
	pull(t, s, user, f.objA, `box = 2`) // 그 사이 상자가 올라간 상태에서 다시 배우기
	if _, err := s.Relearn(ctx, user, f.objA); err != nil {
		t.Fatal(err)
	}
	disp := disputeA(t, s, fake, user, id, judged.ID)
	if !disp.Grade.Conflict || !strings.Contains(disp.Text, "상자는 그대로") {
		t.Fatalf("a dispute after relearn must be a conflict: %+v", disp)
	}
	if a := masteryOf(t, s, user, f.conceptID, f.objA); a.Box != 0 || a.NextReviewAt != nil {
		t.Fatalf("the relearn reset stays: %+v", a)
	}
}

// TestRecommendedLectureMinutes: a recommendation whose next lecture already
// exists shows that lecture's time (section sum); resume items and concepts
// without a lecture are left to the default.
func TestRecommendedLectureMinutes(t *testing.T) {
	s, _, user, f := checkService(t)
	ctx := context.Background()
	items := []learning.Recommendation{
		{Kind: learning.RecEssentialGap, ConceptID: f.conceptID},
		{Kind: learning.RecResume, ConceptID: "resume.only"},
		{Kind: learning.RecNewGoal, ConceptID: "no.such.concept"},
	}
	got := s.recommendedLectureMinutes(ctx, user, items)
	lec, err := s.Store.GetLecture(ctx, user, f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[f.conceptID] != lectureMinutes(lec) || got[f.conceptID] <= 0 {
		t.Fatalf("minutes = %v, want %s: %d", got, f.conceptID, lectureMinutes(lec))
	}
}
