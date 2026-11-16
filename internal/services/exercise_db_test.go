package services

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/internal/store"
)

// TestExerciseAttempts: an answer is saved pending and graded in the
// background, a skip is saved at once, a changed block is refused, the latest
// attempt per block is listed, misses flag the section, and deleting works.
func TestExerciseAttempts(t *testing.T) {
	s, fake, user, f := checkService(t)
	ctx := context.Background()
	content := `{"schemaVersion":1,"title":"페이지 캐시","sections":[
		{"anchor":"principles","kind":"principles","title":"원리","blocks":[
			{"anchor":"principles-1","type":"exercise","text":"4KB 페이지 3장은 몇 바이트?"},
			{"anchor":"principles-2","type":"answer","text":"12288 바이트"},
			{"anchor":"principles-3","type":"exercise","text":"8KB 는?"},
			{"anchor":"principles-4","type":"answer","text":"8192"}]},
		{"anchor":"walkthrough","kind":"walkthrough","title":"따라가기","blocks":[
			{"anchor":"walkthrough-1","type":"lab","kind":"macos","text":"$ vm_stat\n예상: Pages free: …"}]}]}`
	if _, err := testDB(s).ExecContext(ctx, `UPDATE lectures SET content = ? WHERE id = ?`, content, f.lectureID); err != nil {
		t.Fatal(err)
	}
	lec, err := s.Store.GetLecture(ctx, user, f.lectureID)
	if err != nil {
		t.Fatal(err)
	}
	h1 := BlockContentHash("4KB 페이지 3장은 몇 바이트?", "12288 바이트")

	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "principles-1", ContentHash: "stale", Answer: "x"}); !errors.Is(err, ErrExerciseStale) {
		t.Fatalf("changed block must be refused, got %v", err)
	}
	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "principles-2", ContentHash: h1, Answer: "x"}); !errors.Is(err, ErrExerciseNotFound) {
		t.Fatalf("an answer block is not an exercise, got %v", err)
	}
	if _, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "principles-1", ContentHash: h1, Answer: "  "}); !errors.Is(err, ErrInvalidCheck) {
		t.Fatalf("empty answer must be refused, got %v", err)
	}

	fake.push("tutor", `{"verdict":"incorrect","feedback":"3장을 곱했는지 확인해 보세요."}`)
	a, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "principles-1", ContentHash: h1, Answer: "4096"})
	if err != nil || a.Status != store.AttemptPending || a.Kind != learning.AttemptExercise {
		t.Fatalf("submit = %+v %v", a, err)
	}
	graded := waitAttempt(t, s, user, a.ID)
	if graded.Status != store.AttemptDone || graded.Verdict != "incorrect" || graded.Feedback == "" {
		t.Fatalf("graded = %+v", graded)
	}
	skip, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "principles-3", ContentHash: BlockContentHash("8KB 는?", "8192"), Skip: true})
	if err != nil || skip.Status != store.AttemptDone || skip.Verdict != learning.ExerciseSkipped {
		t.Fatalf("skip = %+v %v", skip, err)
	}
	fake.push("tutor", `{"verdict":"matched","feedback":"Pages free 줄을 보면 됩니다."}`)
	lab, err := s.SubmitExercise(ctx, user, lec, ExerciseInput{Anchor: "walkthrough-1", ContentHash: BlockContentHash("$ vm_stat\n예상: Pages free: …", ""),
		Answer: "Pages free: 12345.\n"})
	if err != nil || lab.Kind != learning.AttemptLab {
		t.Fatalf("lab = %+v %v", lab, err)
	}
	if got := waitAttempt(t, s, user, lab.ID); got.Verdict != learning.LabMatched {
		t.Fatalf("lab graded = %+v", got)
	}

	list, err := s.Store.LatestExerciseAttempts(ctx, user, f.lectureID)
	if err != nil || len(list) != 3 {
		t.Fatalf("latest attempts = %d %v", len(list), err)
	}
	fixes, err := s.LectureFixes(ctx, user, f.lectureID)
	if err != nil || len(fixes) != 1 || fixes[0].Section != "principles" {
		raw, _ := json.Marshal(fixes)
		t.Fatalf("fixes = %s %v", raw, err)
	}
	if err := s.Store.DeleteExerciseAttempt(ctx, user, lab.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Store.DeleteExerciseAttempt(ctx, user, "not-a-uuid"); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("bad id → not found, got %v", err)
	}
}

func waitAttempt(t *testing.T, s *Service, user, id string) models.ExerciseAttempt {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for {
		a, err := s.Store.GetExerciseAttempt(context.Background(), user, id)
		if err != nil {
			t.Fatal(err)
		}
		if a.Status != store.AttemptPending || time.Now().After(deadline) {
			return a
		}
		time.Sleep(20 * time.Millisecond)
	}
}
