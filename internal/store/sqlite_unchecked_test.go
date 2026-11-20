package store

import (
	"context"
	"testing"
	"time"
)

// 끝낸 강의는 확인하기를 마치기 전까지 '오늘'의 확인하기 후보로 남고, 다시 만든 강의는 새 버전을 가리킨다.
func TestUncheckedLectures(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	cA, _ := twoConcepts(t, s)
	unchecked := func() []string {
		d, err := s.LoadToday(ctx, user, time.Now())
		if err != nil {
			t.Fatalf("load today: %v", err)
		}
		var ids []string
		for _, u := range d.Unchecked {
			ids = append(ids, u.LectureID)
		}
		return ids
	}

	oldID := rawLecture(t, s, user, cA, "옛 강의", "")
	if err := s.UpdateProgress(ctx, user, oldID, "principles", 40); err != nil {
		t.Fatalf("progress: %v", err)
	}
	if got := unchecked(); len(got) != 0 {
		t.Fatalf("unfinished lecture listed: %v", got)
	}
	if _, err := s.c().exec(ctx, `UPDATE user_lecture_progress SET completed_at = NOW(), percent = 100 WHERE user_id = $1`, user); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if got := unchecked(); len(got) != 1 || got[0] != oldID {
		t.Fatalf("unchecked = %v, want [%s]", got, oldID)
	}

	newID := rawLecture(t, s, user, cA, "새 강의", "")
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: user, ParentLecture: oldID, ChildLectureID: newID, ChildConceptID: cA, Kind: LinkRegeneratedKind}); err != nil {
		t.Fatalf("link new: %v", err)
	}
	if got := unchecked(); len(got) != 1 || got[0] != newID {
		t.Fatalf("unchecked after regenerate = %v, want [%s]", got, newID)
	}

	// 열려 있는(끝나지 않은) 확인은 아직 후보, 새 버전의 확인을 마치면 빠진다.
	if _, err := s.c().exec(ctx, `INSERT INTO check_sessions (id, user_id, kind, concept_id, lecture_id, session_key, status)
		VALUES ('cs-test', $1, 'check', $2, $3, $4, 'active')`, user, cA, newID, "lecture:"+newID); err != nil {
		t.Fatalf("open check: %v", err)
	}
	if got := unchecked(); len(got) != 1 {
		t.Fatalf("open check removed the card: %v", got)
	}
	if _, err := s.c().exec(ctx, `UPDATE check_sessions SET status = 'finished', finished_at = NOW() WHERE user_id = $1`, user); err != nil {
		t.Fatalf("finish check: %v", err)
	}
	if got := unchecked(); len(got) != 0 {
		t.Fatalf("checked lecture still listed: %v", got)
	}
}

// 이 사용자가 열 수 없는 강의(예전 다른 계정으로 만든 즉석 강의)는 학습 목록에서 빠지듯 '오늘'의
// 확인하기·이어서 보기 후보에서도 빠진다 — 눌렀을 때 '요청한 항목을 찾을 수 없어요'(404)가 나지 않게.
func TestTodaySkipsLecturesTheUserCannotOpen(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	cA, cB := twoConcepts(t, s)

	finished := rawLecture(t, s, "old-identity", cA, "옛 계정 강의", "")
	started := rawLecture(t, s, "old-identity", cB, "옛 계정 이어보기", "")
	for _, id := range []string{finished, started} {
		if _, err := s.c().exec(ctx, `INSERT INTO user_lecture_progress (user_id, lecture_id, anchor, percent, last_seen_at)
			VALUES ($1, $2, 'principles', 40, NOW())`, user, id); err != nil {
			t.Fatalf("progress: %v", err)
		}
	}
	if _, err := s.c().exec(ctx, `UPDATE user_lecture_progress SET completed_at = NOW(), percent = 100 WHERE user_id = $1 AND lecture_id = $2`, user, finished); err != nil {
		t.Fatalf("complete: %v", err)
	}
	if _, err := s.GetLecture(ctx, user, finished); err == nil {
		t.Fatal("precondition: the old identity's lecture should not open for this user")
	}

	d, err := s.LoadToday(ctx, user, time.Now())
	if err != nil {
		t.Fatalf("load today: %v", err)
	}
	for _, u := range d.Unchecked {
		if u.LectureID == finished {
			t.Fatalf("a lecture the user cannot open is a 확인하기 candidate: %v", d.Unchecked)
		}
	}
	for _, r := range d.Resume {
		if r.LectureID == started {
			t.Fatalf("a lecture the user cannot open is offered to resume: %v", d.Resume)
		}
	}

	// Their own lecture still is.
	own := rawLecture(t, s, user, cA, "내 강의", "")
	if _, err := s.c().exec(ctx, `INSERT INTO user_lecture_progress (user_id, lecture_id, anchor, percent, last_seen_at, completed_at)
		VALUES ($1, $2, 'principles', 100, NOW(), NOW())`, user, own); err != nil {
		t.Fatalf("own progress: %v", err)
	}
	d, err = s.LoadToday(ctx, user, time.Now())
	if err != nil {
		t.Fatalf("load today: %v", err)
	}
	if len(d.Unchecked) != 1 || d.Unchecked[0].LectureID != own {
		t.Fatalf("unchecked = %v, want only the user's own lecture", d.Unchecked)
	}
}
