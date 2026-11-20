package store

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/migrations"
)

func masteryFixture(t *testing.T, s *SQLiteStore, user string) (conceptID, objectiveID string) {
	t.Helper()
	ctx := context.Background()
	conceptID = "test.mastery-store"
	objectiveID = conceptID + "#1"
	if _, err := s.c().exec(ctx, `INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES ($1, '캐시', 'os', 1, $2)`, conceptID, user); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, '읽기 경로', 1, 1)`, objectiveID, conceptID); err != nil {
		t.Fatal(err)
	}
	return conceptID, objectiveID
}

func saveTestMastery(t *testing.T, s *SQLiteStore, user, objectiveID string, m learning.Mastery) time.Time {
	t.Helper()
	var updated time.Time
	if err := s.tx(context.Background(), func(c conn) error {
		var err error
		updated, err = saveMastery(context.Background(), c, user, objectiveID, "llm", m, "")
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return updated
}

// A changed objective text retires its items and marks the evidence stale
// (box ≤ 1, no integrative credit); unchanged rows are left alone.
func TestObjectiveChangeRetiresItemsAndStalesMastery(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 3, GradStep: 1, Best: learning.BestUnaided, IntegrativeOK: true})
	if _, err := s.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt, objective_text, objective_depth)
		VALUES ($1, $2, $3, 'check', 'recall', '왜?', '읽기 경로', 1)`, NewID(), conceptID, objectiveID); err != nil {
		t.Fatal(err)
	}
	run := func() {
		if err := s.tx(ctx, func(c conn) error { return markObjectiveChanges(ctx, c) }); err != nil {
			t.Fatal(err)
		}
	}
	run()
	list, err := s.ConceptMastery(ctx, user, conceptID)
	if err != nil || len(list) != 1 || list[0].Stale || list[0].Box != 3 {
		t.Fatalf("unchanged objective must keep its evidence: %+v %v", list, err)
	}
	if _, err := s.c().exec(ctx, `UPDATE objectives SET text = '읽기 경로와 readahead' WHERE id = $1`, objectiveID); err != nil {
		t.Fatal(err)
	}
	run()
	list, _ = s.ConceptMastery(ctx, user, conceptID)
	if m := list[0]; !m.Stale || m.Box != 1 || m.IntegrativeOK || m.GradStep != 0 || m.Level != learning.LevelAlone {
		t.Fatalf("changed objective = %+v", m)
	}
	var status string
	if err := s.c().row(ctx, `SELECT status FROM check_items WHERE objective_id = $1`, objectiveID).Scan(&status); err != nil || status != "retired" {
		t.Fatalf("item status = %s (%v)", status, err)
	}
}

// '다시 배우기': the objective's read state is removed, the concept goes back to
// in_progress even from completed, mastery drops to box 0 / lapses 0 / no review, best kept.
func TestReopenObjective(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	if _, err := s.SetConceptState(ctx, user, conceptID, learning.Completed); err != nil {
		t.Fatal(err)
	}
	saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 1, Best: learning.BestUnaided, Lapses: 2,
		NextReviewAt: ptrNow()})
	got, err := s.ReopenObjective(ctx, user, objectiveID)
	if err != nil || got != conceptID {
		t.Fatal(got, err)
	}
	var state string
	var objectiveRows int
	_ = s.c().row(ctx, `SELECT state FROM user_concept_states WHERE user_id = $1 AND concept_id = $2`, user, conceptID).Scan(&state)
	_ = s.c().row(ctx, `SELECT COUNT(*) FROM user_objective_states WHERE user_id = $1 AND objective_id = $2`, user, objectiveID).Scan(&objectiveRows)
	if state != learning.InProgress || objectiveRows != 0 {
		t.Fatalf("concept state %s, objective rows %d", state, objectiveRows)
	}
	list, _ := s.ConceptMastery(ctx, user, conceptID)
	if m := list[0]; m.Box != 0 || m.Lapses != 0 || m.NextReviewAt != nil || m.Best != learning.BestUnaided {
		t.Fatalf("reopened mastery = %+v", m)
	}
	if _, err := s.ReopenObjective(ctx, "someone-else", objectiveID); err != ErrNotFound {
		t.Fatalf("another user's chapter objective must be invisible: %v", err)
	}
}

// Partial settings update keeps the other fields; defaults on first save.
func TestUpdateLearningSettingsPartial(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	def, err := s.GetLearningSettings(ctx, user)
	if err != nil || def.MasteryTarget != 4 || len(def.ReviewIntervals) != 3 || def.ReviewMinutes != 10 || def.ReviewGrader != "self" {
		t.Fatalf("defaults = %+v (%v)", def, err)
	}
	target, grader := 5, "llm"
	st, err := s.UpdateLearningSettings(ctx, user, LearningSettingsPatch{MasteryTarget: &target, ReviewIntervals: []int{2, 9}})
	if err != nil || st.MasteryTarget != 5 || len(st.ReviewIntervals) != 2 || st.ReviewIntervals[1] != 9 || st.ExampleLanguage != "typescript" || st.DailyMinutes != 30 {
		t.Fatalf("first patch = %+v (%v)", st, err)
	}
	st, err = s.UpdateLearningSettings(ctx, user, LearningSettingsPatch{ReviewGrader: &grader})
	if err != nil || st.ReviewGrader != "llm" || st.MasteryTarget != 5 || st.ReviewIntervals[0] != 2 || st.ReviewMinutes != 10 {
		t.Fatalf("second patch must keep earlier fields: %+v (%v)", st, err)
	}
	if st, err = s.SaveLearningSettings(ctx, user, "go", 0); err != nil || st.ExampleLanguage != "go" || st.MasteryTarget != 5 {
		t.Fatalf("legacy save = %+v (%v)", st, err)
	}
}

// SQLite replacements of PostgreSQL behaviour: the open-session partial unique
// index (ON CONFLICT … WHERE), the one-pending index mapped to ErrCheckBusy by
// column name, the pending limit without an advisory lock, item order of
// CheckItemsByID (array_position) and clockNow (clock_timestamp) never repeating.
func TestCheckStoreSQLiteSemantics(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	lectureID := rawLecture(t, s, user, conceptID, "캐시", "")

	n := NewCheckSession{UserID: user, Kind: "check", Grader: "llm", ConceptID: conceptID, LectureID: lectureID, SessionKey: "lecture:" + lectureID}
	first, created, err := s.CreateCheckSession(ctx, n)
	if err != nil || !created || first.Status != "preparing" || len(first.ItemIDs) != 0 {
		t.Fatalf("create = %+v %v %v", first, created, err)
	}
	again, created, err := s.CreateCheckSession(ctx, n)
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("open session for the key must be returned: %+v %v %v", again, created, err)
	}

	key := CheckItemKey{LectureID: lectureID, Purpose: check.PurposeCheck}
	job, started, err := s.StartCheckItemsJob(ctx, user, conceptID, key, 3)
	if err != nil || !started || job.IdempotencyKey != key.JobKey() || job.Status != "running" {
		t.Fatalf("items job = %+v %v %v", job, started, err)
	}
	if again, started, err := s.StartCheckItemsJob(ctx, user, conceptID, key, 3); err != nil || started || again.ID != job.ID {
		t.Fatalf("the running job must be reused: %+v %v %v", again, started, err)
	}
	if _, _, err := s.StartCheckItemsJob(ctx, user, conceptID, CheckItemKey{LectureID: lectureID, Purpose: check.PurposeCheck, PromptHash: "h", CreatedFor: user}, 1); !errors.Is(err, ErrCheckBusy) {
		t.Fatalf("pending limit = %v, want ErrCheckBusy", err)
	}
	items, err := s.SaveCheckItems(ctx, key, conceptID, job.ID, "m", "v", []check.Item{
		{ObjectiveID: objectiveID, Purpose: check.PurposeCheck, Format: "recall", Prompt: "1번", Rubric: check.Rubric{Must: []string{"a"}, Answer: "답"}},
		{ObjectiveID: objectiveID, Purpose: check.PurposeReview, Format: "apply", Prompt: "복습"},
		{Purpose: check.PurposeCheck, Format: "integrative", Prompt: "통합", Rubric: check.Rubric{Covers: []string{objectiveID}}},
	})
	if err != nil || len(items) != 3 || items[0].Rubric.Answer != "답" || items[1].ObjectiveID != "" || items[2].Purpose != check.PurposeReview || string(items[0].Material) != "{}" {
		t.Fatalf("saved items = %+v %v", items, err)
	}
	byID, err := s.CheckItemsByID(ctx, user, []string{items[1].ID, items[0].ID})
	if err != nil || len(byID) != 2 || byID[0].ID != items[1].ID || byID[1].ID != items[0].ID {
		t.Fatalf("CheckItemsByID must keep the order of ids: %+v %v", byID, err)
	}

	if _, err := s.ActivateCheckSession(ctx, user, first.ID, []string{items[0].ID, items[1].ID},
		TurnWrite{ItemID: items[0].ID, Role: "tutor", Move: "ask", Text: "1번"}); err != nil {
		t.Fatal(err)
	}
	tutor := func() *TurnWrite {
		return &TurnWrite{ItemID: items[0].ID, Role: "tutor", Status: "pending", Grader: "llm"}
	}
	pending, err := s.StartCheckTurns(ctx, user, first.ID, &TurnWrite{ItemID: items[0].ID, Role: "learner", Move: "answer", Text: "a"}, tutor(), 3, true)
	if err != nil || pending.Seq != 3 || pending.Status != "pending" {
		t.Fatalf("turns = %+v %v", pending, err)
	}
	if _, err := s.StartCheckTurns(ctx, user, first.ID, nil, tutor(), 3, true); !errors.Is(err, ErrCheckBusy) {
		t.Fatalf("second pending turn = %v, want ErrCheckBusy", err)
	}

	outcome, err := s.FinishCheckItem(ctx, ItemFinish{UserID: user, SessionID: first.ID, Item: items[0],
		Result:  learning.ItemResult{FirstVerdict: "correct", FinalVerdict: "correct"},
		Turn:    TurnWrite{ID: pending.ID, Role: "tutor", Move: "next", Text: "맞아요", Verdict: "correct", Grader: "llm"},
		NextAsk: &TurnWrite{ItemID: items[1].ID, Role: "tutor", Move: "ask", Text: "통합"}, Intervals: []int{1, 7, 30}, Now: time.Now()})
	if err != nil || outcome == "" {
		t.Fatalf("finish = %q %v", outcome, err)
	}
	sess, err := s.GetCheckSession(ctx, user, first.ID)
	if err != nil || sess.Cursor != 1 || sess.Status != "active" || !strings.Contains(string(sess.Result), `"items":[{"itemId":"`+items[0].ID) {
		t.Fatalf("session after the item = %+v %s %v", sess, sess.Result, err)
	}

	seen := map[time.Time]bool{}
	for i := 0; i < 50; i++ {
		ts := clockNow()
		if seen[ts] {
			t.Fatalf("clockNow repeated %v", ts)
		}
		seen[ts] = true
	}
	a := saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 1, Best: learning.BestUnaided})
	b := saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 1, Best: learning.BestUnaided})
	if !b.After(a) {
		t.Fatalf("two writes must get distinct updated_at: %v %v", a, b)
	}
}

// v040Migrations is the migration set shipped in v0.4.0 and v0.5.0 (001–010), the releases before 011.
func v040Migrations(t *testing.T) fs.FS { t.Helper(); return migrationsBefore(t, "011") }

// migrationsBefore is the migration set whose file names sort before prefix (an older release's DB).
func migrationsBefore(t *testing.T, prefix string) fs.FS {
	t.Helper()
	m := fstest.MapFS{}
	all, err := fs.Glob(migrations.Files, "*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range all {
		if name >= prefix {
			continue
		}
		raw, err := fs.ReadFile(migrations.Files, name)
		if err != nil {
			t.Fatal(err)
		}
		m[name] = &fstest.MapFile{Data: raw}
	}
	return m
}

// TestUpgradeFromV040 builds a v0.4.0/v0.5.0 database (migrations 001–010 + saved
// settings), then applies 011: the settings keep their values and get the
// mastery defaults (and 012's pretest_enabled on), the check tables work, and a later seed change marks the
// evidence stale through SyncCurriculum.
func TestUpgradeFromV040(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, v040Migrations(t)); err != nil {
		t.Fatalf("v0.4.0 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.tx(ctx, func(c conn) error { return applySeedPre017(ctx, c, preFoundationsSeed(t, seed)) }); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO user_learning_settings (user_id, example_language, daily_minutes) VALUES ($1, 'go', 45)`, testUser); err != nil {
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
	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.ExampleLanguage != "go" || st.DailyMinutes != 45 || st.MasteryTarget != 4 ||
		len(st.ReviewIntervals) != 3 || st.ReviewIntervals[2] != 30 || st.ReviewMinutes != 10 || st.ReviewGrader != "self" || !st.PretestEnabled {
		t.Fatalf("settings after 011 = %+v (%v)", st, err)
	}
	minutes := 20
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReviewMinutes: &minutes}); err != nil || st.ReviewMinutes != 20 || st.DailyMinutes != 45 {
		t.Fatalf("patch after 011 = %+v (%v)", st, err)
	}

	c := seed.Concepts[0]
	var objectiveID string
	if err := s.c().row(ctx, `SELECT id FROM objectives WHERE concept_id = $1 AND text = $2`, c.ID, c.Objectives[0].Text).Scan(&objectiveID); err != nil {
		t.Fatal(err)
	}
	saveTestMastery(t, s, testUser, objectiveID, learning.Mastery{Box: 2, Best: learning.BestUnaided})
	lectureID := rawLecture(t, s, testUser, c.ID, "옛 강의", "")
	if _, _, err := s.CreateCheckSession(ctx, NewCheckSession{UserID: testUser, Kind: "check", Grader: "llm",
		ConceptID: c.ID, LectureID: lectureID, SessionKey: "lecture:" + lectureID}); err != nil {
		t.Fatalf("check session after 011: %v", err)
	}

	changed := *seed
	changed.Version = seed.Version + "-changed"
	changed.Concepts = append([]curriculum.Concept{}, seed.Concepts...)
	changed.Concepts[0].Objectives = append([]curriculum.Objective{}, c.Objectives...)
	changed.Concepts[0].Objectives[0].Text += " (바뀜)"
	if applied, err := s.SyncCurriculum(ctx, &changed); err != nil || !applied {
		t.Fatalf("seed change: %v %v", applied, err)
	}
	list, err := s.ObjectivesMastery(ctx, testUser, []string{objectiveID})
	if err != nil || len(list) != 1 || !list[0].Stale || list[0].Box != 1 {
		t.Fatalf("changed objective after sync = %+v (%v)", list, err)
	}
}

func ptrNow() *time.Time { t := time.Now(); return &t }

// ConceptsMastery reads many concepts in one query; another user's concept is left out.
// (masteryFixture has a fixed id here, so the other user's concept is inserted directly.)
func TestConceptsMasteryBatch(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, objectiveID := masteryFixture(t, s, user)
	saveTestMastery(t, s, user, objectiveID, learning.Mastery{Box: 2, Best: learning.BestUnaided})
	otherConcept := "test.mastery-other"
	if _, err := s.c().exec(ctx, `INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES ($1, '남의 캐시', 'os', 1, 'someone-else')`, otherConcept); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, '읽기 경로', 1, 1)`, otherConcept+"#1", otherConcept); err != nil {
		t.Fatal(err)
	}
	list, err := s.ConceptsMastery(ctx, user, []string{conceptID, otherConcept, "test.missing"})
	if err != nil || len(list) != 1 || list[0].ConceptID != conceptID || list[0].Level != 4 {
		t.Fatalf("batch = %+v (%v)", list, err)
	}
	if list, err := s.ConceptsMastery(ctx, user, nil); err != nil || len(list) != 0 {
		t.Fatalf("empty batch = %+v (%v)", list, err)
	}
}

// ReviewCounts on SQLite (COUNT FILTER → COUNT(CASE…), TEXT timestamps): the
// overdue boundary is the KST day start, not the UTC one; queued rows count
// apart and graduated rows not at all.
func TestReviewCountsKSTBoundary(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	user := testUser
	conceptID, _ := masteryFixture(t, s, user)
	kst := time.FixedZone("KST", 9*60*60)
	now := time.Date(2026, 11, 13, 8, 30, 0, 0, kst) // 2026-11-12T23:30Z: UTC 날짜는 아직 전날
	dayStart := time.Date(2026, 11, 13, 0, 0, 0, 0, kst)
	rows := []struct {
		due    time.Time
		queued bool
		step   int
	}{
		{dayStart.Add(-time.Millisecond), false, 0}, // 어제(KST) 만기 → overdue
		{dayStart, false, 0},                        // 오늘 0시 만기 → due 만
		{now.Add(time.Hour), false, 0},              // 아직 → 셈 밖
		{dayStart.Add(-time.Hour), true, 0},         // 대기 → queued
		{dayStart.Add(-time.Hour), false, 2},        // 졸업 → 셈 밖
	}
	for i, r := range rows {
		obj := conceptID + "#k" + string(rune('0'+i))
		if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, '경계', 1, $3)`, obj, conceptID, 10+i); err != nil {
			t.Fatal(err)
		}
		if _, err := s.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, grad_step, next_review_at, queued)
			VALUES ($1, $2, $3, 1, $4, $5, $6)`, user, obj, conceptID, r.step, r.due, r.queued); err != nil {
			t.Fatal(err)
		}
	}
	got, err := s.ReviewCounts(ctx, user, now)
	if err != nil || got != (learning.ReviewCounts{Due: 2, Overdue: 1, Queued: 1}) {
		t.Fatalf("counts = %+v (%v)", got, err)
	}
	if got, err := s.ReviewCounts(ctx, "nobody", now); err != nil || got != (learning.ReviewCounts{}) {
		t.Fatalf("no rows → zeros, not NULL: %+v (%v)", got, err)
	}
}
