package store

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/migrations"
)

func TestStudyPrefsDefaultsAndPatch(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	p, err := s.GetStudyPrefs(ctx, testUser)
	if err != nil || p.MemoryModel != learning.MemoryModelFSRS || p.TargetRetention != 0.9 || p.ReminderHour != -1 || len(p.RestDays) != 0 {
		t.Fatalf("defaults = %+v (%v)", p, err)
	}
	r := 0.85
	p, err = s.UpdateStudyPrefs(ctx, testUser, StudyPrefsPatch{TargetRetention: &r, RestDays: []int{0, 6}})
	if err != nil || p.TargetRetention != 0.85 || len(p.RestDays) != 2 || p.MemoryModel != learning.MemoryModelFSRS {
		t.Fatalf("patch = %+v (%v)", p, err)
	}
	hour := 21
	if p, err = s.UpdateStudyPrefs(ctx, testUser, StudyPrefsPatch{ReminderHour: &hour}); err != nil || p.ReminderHour != 21 || len(p.RestDays) != 2 {
		t.Fatalf("second patch keeps the rest = %+v (%v)", p, err)
	}
}

// applyMemory: failures count toward a leech, the FSRS state grows, and the
// first judgement keeps the ladder date (no FSRS history yet).
func TestApplyMemoryLeechAndRetime(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	_, objectiveID := masteryFixture(t, s, testUser)
	now := time.Date(2026, 11, 25, 1, 0, 0, 0, time.UTC)
	ladder := learning.ApplyEvidence(learning.NewMastery(), learning.OutcomeFailed, now, nil)
	run := func(outcome string, at time.Time, m learning.Mastery) learning.Mastery {
		t.Helper()
		var out learning.Mastery
		if err := s.tx(ctx, func(c conn) error {
			var err error
			out, err = applyMemory(ctx, c, testUser, objectiveID, outcome, m, at)
			return err
		}); err != nil {
			t.Fatal(err)
		}
		return out
	}
	first := run(learning.OutcomeFailed, now, ladder)
	if !first.NextReviewAt.Equal(*ladder.NextReviewAt) {
		t.Fatalf("no history keeps the ladder date: %v vs %v", first.NextReviewAt, ladder.NextReviewAt)
	}
	for i := 1; i < learning.LeechThreshold; i++ {
		run(learning.OutcomeAided, now.Add(time.Duration(i)*24*time.Hour), ladder)
	}
	leeches, suspended, err := s.Leeches(ctx, testUser, "ko")
	if err != nil || len(leeches) != 1 || leeches[0].FailCount != learning.LeechThreshold || len(suspended) != 0 {
		t.Fatalf("leeches = %+v / %+v (%v)", leeches, suspended, err)
	}
	// 보류하면 복습 큐에서 빠지고, 풀면 돌아온다.
	if _, err := s.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at)
		SELECT $1, id, concept_id, 1, 'aided', $3 FROM objectives WHERE id = $2`, testUser, objectiveID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	inQueue := func() int {
		q, err := s.ReviewQueue(ctx, testUser, now)
		if err != nil {
			t.Fatal(err)
		}
		return len(q)
	}
	if inQueue() != 1 {
		t.Fatal("due before suspending")
	}
	if err := s.SetLeech(ctx, testUser, objectiveID, LeechSuspend); err != nil || inQueue() != 0 {
		t.Fatalf("suspended objective left the queue: %v", err)
	}
	if _, suspended, _ := s.Leeches(ctx, testUser, "ko"); len(suspended) != 1 {
		t.Fatal("listed as suspended")
	}
	if err := s.SetLeech(ctx, testUser, objectiveID, LeechResume); err != nil || inQueue() != 1 {
		t.Fatalf("resumed objective is back: %v", err)
	}
	if err := s.SetLeech(ctx, testUser, objectiveID, LeechKeep); err != nil {
		t.Fatal(err)
	}
	if leeches, _, _ := s.Leeches(ctx, testUser, "ko"); len(leeches) != 0 {
		t.Fatal("keep clears the mark")
	}
	if err := s.SetLeech(ctx, testUser, "nope#1", LeechKeep); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown objective: %v", err)
	}

	// 기록이 쌓이면 FSRS 간격이 섞인다(같은 상자 날짜에서 달라짐).
	ladderGood := learning.ApplyEvidence(learning.Mastery{Best: learning.BestUnaided, Box: 1}, learning.OutcomeUnaidedCorrect, now.AddDate(0, 0, 10), nil)
	retimed := run(learning.OutcomeUnaidedCorrect, now.AddDate(0, 0, 10), ladderGood)
	if retimed.NextReviewAt.Equal(*ladderGood.NextReviewAt) {
		t.Fatal("with FSRS history the date is blended")
	}
	ladderModel := learning.MemoryModelLadder
	if _, err := s.UpdateStudyPrefs(ctx, testUser, StudyPrefsPatch{MemoryModel: &ladderModel}); err != nil {
		t.Fatal(err)
	}
	if kept := run(learning.OutcomeUnaidedCorrect, now.AddDate(0, 0, 20), ladderGood); !kept.NextReviewAt.Equal(*ladderGood.NextReviewAt) {
		t.Fatal("the ladder model keeps the ladder date")
	}
}

func TestForecastActivityDeadline(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC) // 12:00 KST
	if _, err := s.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at)
		VALUES ($1, $2, $3, 1, 'aided', $4)`, testUser, objectiveID, conceptID, now.AddDate(0, 0, 2)); err != nil {
		t.Fatal(err)
	}
	days, queued, err := s.ReviewForecast(ctx, testUser, now, 14)
	if err != nil || len(days) != 14 || days[2].Objectives != 1 || queued != 0 || days[0].Day != "2026-11-25" {
		t.Fatalf("forecast = %+v queued=%d (%v)", days, queued, err)
	}
	act, err := s.Activity(ctx, testUser, now, 7)
	if err != nil || len(act) != 7 || act[6].Day != "2026-11-25" {
		t.Fatalf("activity = %+v (%v)", act, err)
	}
	goal := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO user_goals (id, user_id, title) VALUES ($1, $2, '캐시')`, goal, testUser); err != nil {
		t.Fatal(err)
	}
	if err := s.SetGoalDeadline(ctx, testUser, goal, "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if due, err := s.GoalDeadline(ctx, testUser, goal); err != nil || due != "2026-12-31" {
		t.Fatalf("deadline = %q (%v)", due, err)
	}
	if all, _ := s.GoalDeadlines(ctx, testUser); all[goal] != "2026-12-31" {
		t.Fatalf("deadlines = %v", all)
	}
	if err := s.SetGoalDeadline(ctx, "other", goal, "2026-12-31"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's goal: %v", err)
	}
	if err := s.SetGoalDeadline(ctx, testUser, goal, ""); err != nil {
		t.Fatal(err)
	}
	if due, _ := s.GoalDeadline(ctx, testUser, goal); due != "" {
		t.Fatal("cleared")
	}
}

func TestReviewCardsFromNote(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 11, 25, 3, 0, 0, 0, time.UTC)
	note := NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'intro', '페이지 캐시', '커널이 디스크 블록을 메모리에 둔다')`, note, testUser); err != nil {
		t.Fatal(err)
	}
	d, err := s.CardDraftFor(ctx, testUser, "note", note)
	if err != nil || d.Question != "페이지 캐시" || d.Answer == "" || d.CardID != "" {
		t.Fatalf("draft = %+v (%v)", d, err)
	}
	if _, err := s.SaveCard(ctx, testUser, "note", note, " ", "x", now); !errors.Is(err, ErrInvalidCard) {
		t.Fatalf("empty question: %v", err)
	}
	c, err := s.SaveCard(ctx, testUser, "note", note, "페이지 캐시는 어디에 있나요?", "커널 메모리", now)
	if err != nil || c.NextReviewAt == nil || !c.NextReviewAt.After(now) {
		t.Fatalf("card = %+v (%v)", c, err)
	}
	if due, _ := s.DueCards(ctx, testUser, now, 10); len(due) != 0 {
		t.Fatal("a new card waits a day")
	}
	tomorrow := now.Add(24 * time.Hour)
	if due, _ := s.DueCards(ctx, testUser, tomorrow, 10); len(due) != 1 {
		t.Fatal("due the next day")
	}
	good, err := s.GradeCard(ctx, testUser, c.ID, learning.RatingGood, tomorrow)
	if err != nil || good.Reps != 1 || !good.NextReviewAt.After(tomorrow.Add(24*time.Hour)) {
		t.Fatalf("graded good = %+v (%v)", good, err)
	}
	again, err := s.GradeCard(ctx, testUser, c.ID, learning.RatingAgain, *good.NextReviewAt)
	if err != nil || again.Lapses != 1 || again.NextReviewAt.Sub(*good.NextReviewAt) > 13*time.Hour {
		t.Fatalf("graded again = %+v (%v)", again, err)
	}
	if due, done, err := s.CardCounts(ctx, testUser, *good.NextReviewAt); err != nil || done != 1 || due != 0 {
		t.Fatalf("counts due=%d done=%d (%v)", due, done, err)
	}
	if d, _ := s.CardDraftFor(ctx, testUser, "note", note); d.CardID != c.ID {
		t.Fatal("draft points at the existing card")
	}
	if err := s.DeleteCard(ctx, "other", c.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("another user's card: %v", err)
	}
	if err := s.DeleteCard(ctx, testUser, c.ID); err != nil {
		t.Fatal(err)
	}
}

func TestSearchIndex(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	c := firstConcept(t, s)
	content, _ := json.Marshal(map[string]any{"title": "페이지 캐시", "sections": []map[string]any{{
		"anchor": "intro", "kind": "concept", "title": "도입",
		"blocks": []map[string]any{{"kind": "prose", "text": "커널은 디스크 블록을 페이지 캐시에 둔다"}},
	}}})
	l := insertTestLecture(t, s, c, "library", "published")
	if _, err := s.c().exec(ctx, `UPDATE lectures SET content = $2, title = '페이지 캐시' WHERE id = $1`, l.ID, string(content)); err != nil {
		t.Fatal(err)
	}
	other := insertTestLecture(t, s, c, "instant", "published")
	if _, err := s.c().exec(ctx, `UPDATE lectures SET content = $2, requested_by = 'someone' WHERE id = $1`, other.ID, string(content)); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ($1, $2, 'intro', '블록', '쓰기 지연')`, NewID(), testUser); err != nil {
		t.Fatal(err)
	}
	hits, err := s.Search(ctx, testUser, "디스크 블록", 10)
	if err != nil || len(hits) != 1 || hits[0].Kind != "lecture" || hits[0].Anchor != "intro" || hits[0].SectionTitle != "도입" || !strings.Contains(hits[0].Snippet, "[[") {
		t.Fatalf("fts hits = %+v (%v)", hits, err)
	}
	short, err := s.Search(ctx, testUser, "블록", 10)
	if err != nil || len(short) != 2 {
		t.Fatalf("short query (LIKE) = %+v (%v)", short, err)
	}
	if none, _ := s.Search(ctx, testUser, `"); DROP`, 10); len(none) != 0 {
		t.Fatal("quotes are a phrase, not syntax")
	}
}

func TestUpgradeFromV022StudyFeatures(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "023")); err != nil {
		t.Fatalf("022 migrate: %v", err)
	}
	conceptID, objectiveID := masteryFixture(t, old, testUser)
	next := time.Date(2026, 12, 1, 0, 0, 0, 0, time.UTC)
	if _, err := old.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, next_review_at)
		VALUES ($1, $2, $3, 2, 'unaided', $4)`, testUser, objectiveID, conceptID, next); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO notes (id, user_id, anchor, quote, body) VALUES ('n1', $1, 'a', '기존 메모', '찾을 수 있다')`, testUser); err != nil {
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
	var l lockedMastery
	if err := s.tx(ctx, func(c conn) error {
		var err error
		l, err = lockMastery(ctx, c, testUser, objectiveID)
		return err
	}); err != nil || l.M.Box != 2 || l.M.NextReviewAt == nil || !l.M.NextReviewAt.Equal(next) {
		t.Fatalf("mastery unchanged by the upgrade: %+v (%v)", l.M, err)
	}
	if p, err := s.GetStudyPrefs(ctx, testUser); err != nil || p.MemoryModel != learning.MemoryModelFSRS {
		t.Fatalf("prefs after upgrade = %+v (%v)", p, err)
	}
	if hits, err := s.Search(ctx, testUser, "기존 메모", 5); err != nil || len(hits) != 1 || hits[0].RefID != "n1" {
		t.Fatalf("existing notes are indexed: %+v (%v)", hits, err)
	}
}
