package store

import (
	"context"
	"path/filepath"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/migrations"
)

// preFoundationsSeed is the shipped seed without the foundations track: what a
// release before 019 could hold (its schema allowed depth 1..3 only).
func preFoundationsSeed(t *testing.T, seed *curriculum.Seed) *curriculum.Seed {
	t.Helper()
	var concepts []curriculum.Concept
	for _, c := range seed.Concepts {
		if c.Area != curriculum.AreaFoundations {
			c.Objectives = append([]curriculum.Objective{}, c.Objectives...)
			concepts = append(concepts, c)
		}
	}
	out, err := curriculum.Build(concepts)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// TestUpgradeFromV018Foundations: 019 rebuilds the four depth-checked tables
// keeping their rows and indexes, then the foundations seed (depth 0) syncs and
// a depth-0 lecture, job and objective state can be stored.
func TestUpgradeFromV018Foundations(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "019")); err != nil {
		t.Fatalf("018 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.SyncCurriculum(ctx, preFoundationsSeed(t, seed)); err != nil {
		t.Fatal(err)
	}
	c := seed.Concepts[0]
	objectiveID := curriculum.ObjectiveID(c.ID, 1)
	lectureID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title, content, published_at, language, reader_level)
		VALUES ($1, $2, $3, $4, 1, 'library', 'published', 1, '옛 강의', '{}', NOW(), 'ko', 'novice')`, lectureID, c.ID, []string{objectiveID}, objectiveID); err != nil {
		t.Fatal(err)
	}
	if _, err := old.SetConceptState(ctx, testUser, c.ID, learning.Known); err != nil {
		t.Fatal(err)
	}
	if err := old.Close(); err != nil {
		t.Fatal(err)
	}

	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("upgrade migrate: %v", err)
	}
	for _, idx := range []string{"idx_concepts_area_status", "idx_concepts_owner", "idx_lectures_concept_status", "idx_lectures_target",
		"idx_lectures_content_hash", "lectures_target_language_idx", "idx_user_objective_states_user"} {
		if !indexExists(t, s, idx) {
			t.Errorf("index %s missing after 019", idx)
		}
	}
	lec, err := s.GetLecture(ctx, testUser, lectureID)
	if err != nil || lec.Title != "옛 강의" || lec.ReaderLevel != learning.ReaderNovice || lec.Depth != 1 {
		t.Fatalf("old lecture after 019 = %+v (%v)", lec, err)
	}
	var known int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM user_objective_states WHERE user_id = $1 AND concept_id = $2 AND state = 'known'`, testUser, c.ID).Scan(&known); err != nil || known == 0 {
		t.Fatalf("objective states kept: %d %v", known, err)
	}

	if _, err := s.SyncCurriculum(ctx, seed); err != nil {
		t.Fatalf("sync with foundations: %v", err)
	}
	found := curriculum.FoundationsTrack[0]
	var depth int
	if err := s.c().row(ctx, `SELECT depth FROM concepts WHERE id = $1`, found).Scan(&depth); err != nil || depth != 0 {
		t.Fatalf("foundations concept depth = %d (%v)", depth, err)
	}
	fo := curriculum.ObjectiveID(found, 1)
	if _, err := s.c().exec(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title, content, published_at)
		VALUES ($1, $2, $3, $4, 0, 'library', 'published', 1, '기초 강의', '{}', NOW())`, NewID(), found, []string{fo}, fo); err != nil {
		t.Fatalf("depth-0 lecture: %v", err)
	}
	if _, err := s.SetConceptState(ctx, testUser, found, learning.Known); err != nil {
		t.Fatalf("depth-0 objective state: %v", err)
	}
	job, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "found-job", Kind: learning.JobInstant, UserID: testUser,
		ConceptID: found, ObjectiveKey: fo, Depth: 0, Reason: "t", ReaderLevel: learning.ReaderBeginner})
	if err != nil || !created {
		t.Fatalf("depth-0 job: %v %v", created, err)
	}
	// 깊이 0 잡도 depth 가 NULL 이 아니라 0 으로 저장돼야 같은 대상 중복 생성이 막힌다.
	if got, err := s.ActiveJobForTarget(ctx, fo, 0, learning.LocaleKO, learning.ReaderBeginner); err != nil || got.ID != job.ID {
		t.Fatalf("active depth-0 job: %v %v", got.ID, err)
	}
}
