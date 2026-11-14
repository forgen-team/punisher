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
