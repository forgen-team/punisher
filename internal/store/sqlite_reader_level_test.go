package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/migrations"
)

// Learner-level ①: reader level and area familiarity settings, and the level
// as part of a lecture/job target (like the language, 017).

func TestReaderLevelSettings(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.ReaderLevel != "" || st.AreaFamiliarity == nil || len(st.AreaFamiliarity) != 0 {
		t.Fatalf("defaults = %q %v (%v)", st.ReaderLevel, st.AreaFamiliarity, err)
	}
	if got := s.ReaderLevelFor(ctx, testUser); got != learning.ReaderPractitioner {
		t.Fatalf("unchosen level generates as practitioner: %q", got)
	}
	// A save of another field keeps the level unchosen and the map empty.
	daily := 45
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{DailyMinutes: &daily}); err != nil ||
		st.ReaderLevel != "" || st.AreaFamiliarity == nil || len(st.AreaFamiliarity) != 0 {
		t.Fatalf("first save of another field = %q %v (%v)", st.ReaderLevel, st.AreaFamiliarity, err)
	}
	level := learning.ReaderBeginner
	fam := map[string]string{"os": learning.FamiliarityCanExplain, "database": learning.FamiliarityHeard}
	st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &level, AreaFamiliarity: fam})
	if err != nil || st.ReaderLevel != level || st.AreaFamiliarity["os"] != learning.FamiliarityCanExplain || len(st.AreaFamiliarity) != 2 || st.DailyMinutes != 45 {
		t.Fatalf("saved = %+v (%v)", st, err)
	}
	if got := s.ReaderLevelFor(ctx, testUser); got != level {
		t.Fatalf("level for = %q", got)
	}
	// nil keeps the map; an empty map clears it.
	adv := learning.ReaderAdvanced
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &adv}); err != nil || len(st.AreaFamiliarity) != 2 {
		t.Fatalf("nil map must keep: %v (%v)", st.AreaFamiliarity, err)
	}
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{AreaFamiliarity: map[string]string{}}); err != nil ||
		len(st.AreaFamiliarity) != 0 || st.ReaderLevel != adv {
		t.Fatalf("empty map must clear: %+v (%v)", st, err)
	}
	// Unknown stored values read as unchosen / dropped (never surface to the web).
	if _, err := s.c().exec(ctx, `UPDATE user_learning_settings SET reader_level = 'expert', area_familiarity = '{"os":"new","nope":"heard","db":"x"}' WHERE user_id = $1`, testUser); err != nil {
		t.Fatal(err)
	}
	if st, err = s.GetLearningSettings(ctx, testUser); err != nil || st.ReaderLevel != "" || len(st.AreaFamiliarity) != 1 || st.AreaFamiliarity["os"] != "new" {
		t.Fatalf("unknown stored values = %q %v (%v)", st.ReaderLevel, st.AreaFamiliarity, err)
	}
}

// Same target + level reuses the lecture; another level does not (LangExact),
// LangPrefer falls back to it; the active-job index is per level.
func TestReaderLevelLectureReuse(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	ids := []string{objectiveID}
	prac, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, testUser))
	if err != nil || prac.ReaderLevel != learning.ReaderPractitioner {
		t.Fatalf("\"\" level saves as practitioner: %q (%v)", prac.ReaderLevel, err)
	}
	for _, level := range []string{"", learning.ReaderPractitioner} {
		if got, err := s.PublishedLectureCovering(ctx, testUser, conceptID, 1, ids, learning.LocaleKO, level, LangExact); err != nil || got.ID != prac.ID {
			t.Fatalf("same level %q must reuse: %v %v", level, got.ID, err)
		}
	}
	if _, err := s.PublishedLectureCovering(ctx, testUser, conceptID, 1, ids, learning.LocaleKO, learning.ReaderBeginner, LangExact); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another level must not reuse: %v", err)
	}
	if _, err := s.LatestLectureForTarget(ctx, learning.JobInstant, conceptID, objectiveID, 1, learning.LocaleKO, learning.ReaderBeginner, []string{"published"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("latest for another level: %v", err)
	}
	if got, err := s.PublishedLectureCovering(ctx, testUser, conceptID, 1, ids, learning.LocaleKO, learning.ReaderBeginner, LangPrefer); err != nil || got.ID != prac.ID {
		t.Fatalf("prefer falls back to another level: %v %v", got.ID, err)
	}
	nl := testLecture(conceptID, objectiveID, learning.LocaleKO, testUser)
	nl.ReaderLevel = learning.ReaderBeginner
	beg, err := s.InsertLecture(ctx, nl)
	if err != nil || beg.ReaderLevel != learning.ReaderBeginner || beg.Version != prac.Version+1 {
		t.Fatalf("beginner lecture = %q v%d (%v)", beg.ReaderLevel, beg.Version, err)
	}
	if got, err := s.LatestLectureForTarget(ctx, learning.JobInstant, conceptID, objectiveID, 1, learning.LocaleKO, learning.ReaderBeginner, []string{"published"}); err != nil || got.ID != beg.ID {
		t.Fatalf("latest beginner: %v %v", got.ID, err)
	}
	if got, err := s.PublishedLectureCovering(ctx, testUser, conceptID, 1, ids, learning.LocaleKO, learning.ReaderPractitioner, LangPrefer); err != nil || got.ID != prac.ID {
		t.Fatalf("prefer takes the own level first: %v %v", got.ID, err)
	}

	newJob := func(level string) NewJob {
		return NewJob{IdempotencyKey: "instant:" + RandomToken(), Kind: learning.JobInstant, UserID: testUser,
			ConceptID: conceptID, ObjectiveKey: objectiveID, Depth: 1, Reason: "test", ReaderLevel: level}
	}
	pj, created, err := s.CreateJob(ctx, newJob(""))
	if err != nil || !created || pj.ReaderLevel != learning.ReaderPractitioner {
		t.Fatalf("practitioner job: %+v %v %v", pj, created, err)
	}
	bj, created, err := s.CreateJob(ctx, newJob(learning.ReaderBeginner))
	if err != nil || !created {
		t.Fatalf("beginner job must run beside the practitioner job: %v %v", created, err)
	}
	if _, _, err := s.CreateJob(ctx, newJob(learning.ReaderPractitioner)); !errors.Is(err, ErrJobActive) {
		t.Fatalf("second practitioner job must be blocked: %v", err)
	}
	if got, err := s.ActiveJobForTarget(ctx, objectiveID, 1, learning.LocaleKO, learning.ReaderBeginner); err != nil || got.ID != bj.ID {
		t.Fatalf("active beginner: %v %v", got.ID, err)
	}
}

// TestUpgradeFromV017 builds a v0.9 database (001–017 with saved settings, a
// lecture and an active job), then applies 018: rows keep their data, old
// lectures and jobs read as practitioner, saved settings have readerLevel ""
// (the first setup shows) and an empty familiarity map.
func TestUpgradeFromV017(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "018")); err != nil {
		t.Fatalf("017 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.SyncCurriculum(ctx, preFoundationsSeed(t, seed)); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO user_learning_settings (user_id, example_language, daily_minutes, locale) VALUES ($1, 'go', 45, 'en')`, testUser); err != nil {
		t.Fatal(err)
	}
	c := seed.Concepts[0]
	objectiveID := curriculum.ObjectiveID(c.ID, 1)
	lectureID := NewID()
	if _, err := old.c().exec(ctx, `INSERT INTO lectures (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title, content, published_at, language)
		VALUES ($1, $2, $3, $4, 1, 'library', 'published', 1, '옛 강의', '{}', NOW(), 'ko')`, lectureID, c.ID, []string{objectiveID}, objectiveID); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, concept_id, objective_key, depth, status, reason, language)
		VALUES ($1, 'old-job', 'instant', $2, $3, $4, 1, 'running', 'old', 'ko')`, NewID(), testUser, c.ID, objectiveID); err != nil {
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
	if err := s.Migrate(ctx, migrations.Files); err != nil {
		t.Fatalf("second migrate (no-op): %v", err)
	}
	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.ReaderLevel != "" || st.AreaFamiliarity == nil || len(st.AreaFamiliarity) != 0 ||
		st.ExampleLanguage != "go" || st.DailyMinutes != 45 || st.Locale != learning.LocaleEN {
		t.Fatalf("settings after 018 = %+v (%v)", st, err)
	}
	lec, err := s.GetLecture(ctx, testUser, lectureID)
	if err != nil || lec.ReaderLevel != learning.ReaderPractitioner || lec.Title != "옛 강의" || lec.Language != learning.LocaleKO {
		t.Fatalf("old lecture after 018 = %q %q (%v)", lec.ReaderLevel, lec.Title, err)
	}
	if got, err := s.PublishedLectureCovering(ctx, testUser, c.ID, 1, []string{objectiveID}, learning.LocaleKO, "", LangExact); err != nil || got.ID != lectureID {
		t.Fatalf("old lecture reused for practitioner: %v %v", got.ID, err)
	}
	if !indexExists(t, s, "generation_jobs_one_active_target") || !indexExists(t, s, "lectures_target_language_idx") {
		t.Fatal("018 indexes missing")
	}
	// The old running job is practitioner: a beginner job of the same target can start, a practitioner one cannot.
	if _, created, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "new-beginner", Kind: learning.JobInstant, UserID: testUser,
		ConceptID: c.ID, ObjectiveKey: objectiveID, Depth: 1, Reason: "t", ReaderLevel: learning.ReaderBeginner}); err != nil || !created {
		t.Fatalf("beginner job beside the old job: %v %v", created, err)
	}
	if _, _, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "new-practitioner", Kind: learning.JobInstant, UserID: testUser,
		ConceptID: c.ID, ObjectiveKey: objectiveID, Depth: 1, Reason: "t"}); !errors.Is(err, ErrJobActive) {
		t.Fatalf("practitioner job must wait for the old job: %v", err)
	}
}
