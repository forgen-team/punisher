package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
)

func conceptState(t *testing.T, s *SQLiteStore, conceptID string) string {
	t.Helper()
	var state string
	err := s.DB().QueryRow(`SELECT state FROM user_concept_states WHERE user_id = ? AND concept_id = ?`, testUser, conceptID).Scan(&state)
	if err != nil {
		return learning.NotStarted
	}
	return state
}

func TestUncompleteLectureRestoresProgress(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	l := insertTestLecture(t, s, c, "library", "published")
	if err := s.UpdateProgress(ctx, testUser, l.ID, "intro", 60); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteLecture(ctx, testUser, l.ID); err != nil {
		t.Fatal(err)
	}
	res, err := s.UncompleteLecture(ctx, testUser, l.ID)
	if err != nil {
		t.Fatalf("uncomplete: %v", err)
	}
	if res.ConceptState != learning.InProgress || conceptState(t, s, c.ID) != learning.InProgress {
		t.Fatalf("concept after undo = %+v / %s, want in_progress", res, conceptState(t, s, c.ID))
	}
	if done, _ := s.DoneSet(ctx, testUser, c.ID); len(done) != 0 {
		t.Fatalf("undo kept %d completed objectives", len(done))
	}
	lec, err := s.GetLecture(ctx, testUser, l.ID)
	if err != nil || lec.Progress == nil || lec.Progress.CompletedAt != nil {
		t.Fatalf("progress after undo = %+v %v", lec.Progress, err)
	}
	// 다시 완료할 수 있다(완료 → 되돌리기 → 완료).
	if again, err := s.CompleteLecture(ctx, testUser, l.ID); err != nil || len(again.ObjectivesCompleted) != 1 {
		t.Fatalf("complete again = %+v %v", again, err)
	}
	if _, err := s.UncompleteLecture(ctx, "someone-else", insertTestLecture(t, s, c, "instant", "published").ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's lecture: %v", err)
	}
}

func TestUncompleteKeepsKnown(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	l := insertTestLecture(t, s, c, "library", "published")
	if _, err := s.SetConceptState(ctx, testUser, c.ID, learning.Known); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CompleteLecture(ctx, testUser, l.ID); err != nil {
		t.Fatal(err)
	}
	if res, err := s.UncompleteLecture(ctx, testUser, l.ID); err != nil || res.ConceptState != learning.Known {
		t.Fatalf("known must stay known: %+v %v", res, err)
	}
}

func TestUndoKnownBackToPrevious(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)

	if _, err := s.SetConceptState(ctx, testUser, c.ID, learning.Known); err != nil {
		t.Fatal(err)
	}
	state, err := s.UndoKnown(ctx, testUser, c.ID, learning.NotStarted)
	if err != nil || state != learning.NotStarted || conceptState(t, s, c.ID) != learning.NotStarted {
		t.Fatalf("undo to not_started = %s %v", state, err)
	}
	if done, _ := s.DoneSet(ctx, testUser, c.ID); len(done) != 0 {
		t.Fatalf("undo kept %d known objectives", len(done))
	}

	l := insertTestLecture(t, s, c, "library", "published")
	if err := s.UpdateProgress(ctx, testUser, l.ID, "intro", 30); err != nil {
		t.Fatal(err)
	}
	if _, err := s.SetConceptState(ctx, testUser, c.ID, learning.Known); err != nil {
		t.Fatal(err)
	}
	if state, err := s.UndoKnown(ctx, testUser, c.ID, learning.InProgress); err != nil || state != learning.InProgress || conceptState(t, s, c.ID) != learning.InProgress {
		t.Fatalf("undo to in_progress = %s %v", state, err)
	}
	// 이미 앎이 아니면 바꾸지 않는다(두 번 눌러도 안전).
	if state, err := s.UndoKnown(ctx, testUser, c.ID, learning.NotStarted); err != nil || state != learning.InProgress {
		t.Fatalf("undo when not known = %s %v", state, err)
	}
	if _, err := s.UndoKnown(ctx, testUser, c.ID, learning.Known); !errors.Is(err, learning.ErrInvalidState) {
		t.Fatalf("previous=known must be rejected: %v", err)
	}
}

func TestResumeCheckSession(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	sess, _, err := s.CreateCheckSession(ctx, NewCheckSession{UserID: testUser, Kind: "check", Grader: "llm", ConceptID: c.ID, SessionKey: "lecture:x"})
	if err != nil {
		t.Fatal(err)
	}
	set := func(sql string, args ...any) {
		t.Helper()
		if _, err := s.DB().Exec(sql, args...); err != nil {
			t.Fatal(err)
		}
	}
	set(`UPDATE check_sessions SET item_ids = '["a","b"]', cursor = 1, result = '{"items":[{"itemId":"a"}]}' WHERE id = ?`, sess.ID)
	if err := s.SetCheckSessionStatus(ctx, testUser, sess.ID, []string{"preparing"}, "abandoned", ""); err != nil {
		t.Fatal(err)
	}
	since := time.Now().Add(-10 * time.Minute)
	if err := s.ResumeCheckSession(ctx, testUser, sess.ID, since); err != nil {
		t.Fatalf("resume: %v", err)
	}
	got, err := s.GetCheckSession(ctx, testUser, sess.ID)
	if err != nil || got.Status != "active" || got.FinishedAt != nil || got.Cursor != 1 {
		t.Fatalf("resumed = %+v %v", got, err)
	}
	if err := s.ResumeCheckSession(ctx, testUser, sess.ID, since); !errors.Is(err, ErrCheckClosed) {
		t.Fatalf("an active session is not resumable: %v", err)
	}

	// 반쯤 답한 문항을 닫으며 그만둔 세션(지금 문항의 결과가 이미 있음)은 되돌리지 않는다.
	set(`UPDATE check_sessions SET status = 'abandoned', finished_at = ?, result = '{"items":[{"itemId":"a"},{"itemId":"b"}]}' WHERE id = ?`, FormatTime(time.Now()), sess.ID)
	if err := s.ResumeCheckSession(ctx, testUser, sess.ID, since); !errors.Is(err, ErrCheckClosed) {
		t.Fatalf("closed current item must stay closed: %v", err)
	}
	// 오래전에 그만둔 세션도 되돌리지 않는다.
	set(`UPDATE check_sessions SET result = '{"items":[{"itemId":"a"}]}', finished_at = ? WHERE id = ?`, FormatTime(time.Now().Add(-time.Hour)), sess.ID)
	if err := s.ResumeCheckSession(ctx, testUser, sess.ID, since); !errors.Is(err, ErrCheckClosed) {
		t.Fatalf("an old abandon must stay: %v", err)
	}
	// 같은 키로 새 세션을 열었으면 겹치므로 busy.
	set(`UPDATE check_sessions SET finished_at = ? WHERE id = ?`, FormatTime(time.Now()), sess.ID)
	if _, _, err := s.CreateCheckSession(ctx, NewCheckSession{UserID: testUser, Kind: "check", Grader: "llm", ConceptID: c.ID, SessionKey: "lecture:x"}); err != nil {
		t.Fatal(err)
	}
	if err := s.ResumeCheckSession(ctx, testUser, sess.ID, since); !errors.Is(err, ErrCheckBusy) {
		t.Fatalf("another open session of the key: %v", err)
	}
}
