package store

import (
	"context"
	"testing"
	"time"
)

// 다시 만든 강의가 공개되면 이어보기는 옛 강의 대신 새 버전을 가리키고, 새 버전을 읽기 시작하면 옛 줄은 사라진다.
func TestResumePointsToRegeneratedVersion(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	cA, _ := twoConcepts(t, s)
	oldID := rawLecture(t, s, user, cA, "옛 강의", "")
	if err := s.UpdateProgress(ctx, user, oldID, "principles", 8); err != nil {
		t.Fatalf("progress: %v", err)
	}
	resumeIDs := func() []string {
		d, err := s.LoadToday(ctx, user, time.Now())
		if err != nil {
			t.Fatalf("load today: %v", err)
		}
		var ids []string
		for _, r := range d.Resume {
			ids = append(ids, r.LectureID)
		}
		return ids
	}

	draftID := rawLecture(t, s, user, cA, "작성 중", "")
	if _, err := s.c().exec(ctx, `UPDATE lectures SET status = 'draft' WHERE id = $1`, draftID); err != nil {
		t.Fatal(err)
	}
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: user, ParentLecture: oldID, ChildLectureID: draftID, ChildConceptID: cA, Kind: LinkRegeneratedKind}); err != nil {
		t.Fatalf("link draft: %v", err)
	}
	if got := resumeIDs(); len(got) != 1 || got[0] != oldID {
		t.Fatalf("resume with unpublished version = %v, want [%s]", got, oldID)
	}

	newID := rawLecture(t, s, user, cA, "새 강의", "")
	if err := s.LinkDeeper(ctx, DeeperLink{UserID: user, ParentLecture: oldID, ChildLectureID: newID, ChildConceptID: cA, Kind: LinkRegeneratedKind}); err != nil {
		t.Fatalf("link new: %v", err)
	}
	if got := resumeIDs(); len(got) != 1 || got[0] != newID {
		t.Fatalf("resume after regenerate = %v, want [%s]", got, newID)
	}

	if err := s.UpdateProgress(ctx, user, newID, "situation_question", 5); err != nil {
		t.Fatalf("progress new: %v", err)
	}
	if got := resumeIDs(); len(got) != 1 || got[0] != newID {
		t.Fatalf("resume once the new version is read = %v, want [%s] only", got, newID)
	}
}
