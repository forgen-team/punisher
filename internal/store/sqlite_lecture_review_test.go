package store

import (
	"context"
	"io/fs"
	"path/filepath"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
	"github.com/wooo-jin/punisher/migrations"
)

const upgradeLectureContent = `{"schemaVersion":1,"title":"캐시","sections":[
	{"anchor":"s1","kind":"principles","title":"원리","minutes":40,"blocks":[{"anchor":"s1-b1","type":"markdown","text":"읽은 블록은 메모리에 남는다"}]}]}`

// reopenAt closes s and opens the same file migrated to the next release:
// the migrations before `before` ("" = all of them).
func reopenAt(t *testing.T, s *SQLiteStore, path, before string) *SQLiteStore {
	t.Helper()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	next, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	files := fs.FS(migrations.Files)
	if before != "" {
		files = migrationsBefore(t, before)
	}
	if err := next.Migrate(context.Background(), files); err != nil {
		t.Fatalf("migrate (before %q): %v", before, err)
	}
	return next
}

func indexExists(t *testing.T, s *SQLiteStore, name string) bool {
	t.Helper()
	var n int
	if err := s.c().row(context.Background(), `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = $1`, name).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n == 1
}

// TestUpgradeFromV013 builds a database at 013 (the P1–P4 mastery branch: a
// mastery row, a judged turn and a published lecture whose estimated_minutes
// is the old section sum), then applies 014 and 015 one release at a time:
// 014 adds the empty evidence marker and the check_turns indexes (the old
// turn's misconception can still be acknowledged, '다시 배우기' moves the
// marker); 015 recomputes the old lecture's minutes from its content, puts it
// in the backfill queue (no pass review) and keeps a new lecture's estimate.
func TestUpgradeFromV013(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "014")); err != nil {
		t.Fatalf("013 migrate: %v", err)
	}
	conceptID, objectiveID := masteryFixture(t, old, testUser)
	turnID := judgedTurn(t, old, testUser, conceptID, objectiveID, "항상 디스크에서 읽는다")
	if _, err := old.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, lapses)
		VALUES ($1, $2, $3, 2, 'unaided', 0)`, testUser, objectiveID, conceptID); err != nil {
		t.Fatal(err)
	}
	lectureID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, title, content,
		estimated_minutes, published_at) VALUES ($1, $2, $3, $4, 1, 'library', 'published', '캐시', $5, 40, NOW())`,
		lectureID, conceptID, []string{objectiveID}, objectiveID, upgradeLectureContent); err != nil {
		t.Fatal(err)
	}

	// 013 → 014
	s := reopenAt(t, old, path, "015")
	if !indexExists(t, s, "check_turns_user_item_idx") || !indexExists(t, s, "check_turns_user_created_idx") {
		t.Fatal("014 indexes missing")
	}
	var l lockedMastery
	if err := s.tx(ctx, func(c conn) error {
		var err error
		l, err = lockMastery(ctx, c, testUser, objectiveID)
		return err
	}); err != nil || !l.Exists || l.Mark != "" || l.M.Box != 2 {
		t.Fatalf("mastery after 014 = %+v (%v)", l, err)
	}
	if m, created, err := s.AcknowledgeMisconception(ctx, testUser, turnID); err != nil || !created || m.Text != "항상 디스크에서 읽는다" {
		t.Fatalf("old turn acknowledged after 014 = %+v %v (%v)", m, created, err)
	}
	if _, err := s.ReopenObjective(ctx, testUser, objectiveID); err != nil {
		t.Fatal(err)
	}
	if err := s.tx(ctx, func(c conn) error {
		var err error
		l, err = lockMastery(ctx, c, testUser, objectiveID)
		return err
	}); err != nil || l.Mark == "" || l.M.Box != 0 {
		t.Fatalf("relearn moves the marker: %+v (%v)", l, err)
	}

	// 014 → 015
	s = reopenAt(t, s, path, "")
	t.Cleanup(func() { _ = s.Close() })
	var applied bool
	if err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '015_lecture_quality.sql')`).Scan(&applied); err != nil || !applied {
		t.Fatalf("015 applied = %v (%v)", applied, err)
	}
	if !indexExists(t, s, "lecture_reviews_pass_idx") {
		t.Fatal("015 index missing")
	}
	queue, err := s.LecturesToReview(ctx, 10)
	if err != nil || len(queue) != 1 || queue[0].ID != lectureID {
		t.Fatalf("the old lecture waits for the backfill review: %+v (%v)", queue, err)
	}
	if got := queue[0]; got.ReviewState != models.ReviewUnreviewed || got.EstimatedMinutes == 40 || got.EstimatedMinutes < 1 {
		t.Fatalf("old lecture: review=%s minutes=%d (recomputed from the content, not the section sum)", got.ReviewState, got.EstimatedMinutes)
	}
	if err := s.RequestLectureReview(ctx, lectureID); err != nil {
		t.Fatal(err)
	}
	var requested bool
	if err := s.c().row(ctx, `SELECT review_requested_at IS NOT NULL FROM lectures WHERE id = $1`, lectureID).Scan(&requested); err != nil || !requested {
		t.Fatalf("review requested = %v (%v)", requested, err)
	}
	if err := s.InsertReview(ctx, lectureID, models.LectureReview{Verdict: "pass", Reviewer: "backfill", Attempt: 1}); err != nil {
		t.Fatal(err)
	}
	if err := s.SetLectureReviewOutcome(ctx, lectureID, true, "ignored"); err != nil {
		t.Fatal(err)
	}
	if queue, err := s.LecturesToReview(ctx, 10); err != nil || len(queue) != 0 {
		t.Fatalf("a passed lecture leaves the queue: %+v (%v)", queue, err)
	}

	fresh, err := s.InsertLecture(ctx, NewLecture{ConceptID: conceptID, ObjectiveIDs: []string{objectiveID}, ObjectiveKey: objectiveID,
		Depth: 1, Kind: learning.JobLibrary, Status: "published", Title: "새 강의", Content: []byte(upgradeLectureContent), EstimatedMinutes: 17})
	if err != nil || fresh.EstimatedMinutes != 17 || fresh.Version != 2 {
		t.Fatalf("a new lecture keeps its computed estimate: %+v (%v)", fresh, err)
	}
	if _, err := s.LoadToday(ctx, testUser, time.Now()); err != nil {
		t.Fatalf("today after 015: %v", err)
	}
}
