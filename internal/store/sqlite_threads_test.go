package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/migrations"
)

// Reading threads (learner-level ④): quote/parent/mode round-trip through
// create, list, thread and history; plain questions default to a question root.
func TestQuestionThreadColumns(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	lec, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, testUser))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Kind: "question", Question: "q"})
	if err != nil || plain.Mode != learning.ThreadQuestion || plain.Quote != "" || plain.ParentID != "" {
		t.Fatalf("plain question = %+v (%v)", plain, err)
	}
	root, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Anchor: "s1", Kind: "question",
		Question: "더 쉽게", Quote: "고른 구절", Mode: learning.ThreadEasier})
	if err != nil || root.Quote != "고른 구절" || root.Mode != learning.ThreadEasier || root.ParentID != "" {
		t.Fatalf("root = %+v (%v)", root, err)
	}
	child, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Anchor: "s1", Kind: "question",
		Question: "왜?", ParentID: root.ID, Mode: learning.ThreadDeeper})
	if err != nil || child.ParentID != root.ID || child.Mode != learning.ThreadDeeper {
		t.Fatalf("child = %+v (%v)", child, err)
	}
	thread, err := s.ListThread(ctx, testUser, root.ID)
	if err != nil || len(thread) != 2 || thread[0].ID != root.ID || thread[1].ID != child.ID {
		t.Fatalf("thread = %+v (%v)", thread, err)
	}
	if _, err := s.ListThread(ctx, "someone-else", root.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("another user's thread: %v, want ErrNotFound", err)
	}
	list, err := s.ListQuestions(ctx, testUser, lec.ID)
	if err != nil || len(list) != 3 || list[1].Quote != "고른 구절" || list[2].ParentID != root.ID {
		t.Fatalf("list = %+v (%v)", list, err)
	}
	mine, _, err := s.ListMyQuestions(ctx, testUser, "", 10, "")
	byID := map[string]models.MyQuestion{}
	for _, q := range mine {
		byID[q.ID] = q
	}
	if err != nil || len(mine) != 3 || byID[child.ID].ParentID != root.ID || byID[root.ID].Mode != learning.ThreadEasier {
		t.Fatalf("my questions = %+v (%v)", mine, err)
	}
	if n, err := s.EasierRequestsSince(ctx, testUser, time.Now().Add(-time.Hour)); err != nil || n != 0 {
		t.Fatalf("a pending easier entry must not count: %d (%v)", n, err)
	}
	if _, err := s.AnswerQuestion(ctx, root.ID, "답", "m", nil, nil); err != nil {
		t.Fatal(err)
	}
	failed, err := s.CreateQuestion(ctx, testUser, models.Question{LectureID: lec.ID, Kind: "question",
		Question: "더 쉽게", Quote: "다른 구절", Mode: learning.ThreadEasier})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.FailQuestion(ctx, failed.ID, "m", "budget"); err != nil {
		t.Fatal(err)
	}
	if n, err := s.EasierRequestsSince(ctx, testUser, time.Now().Add(-time.Hour)); err != nil || n != 1 {
		t.Fatalf("only the answered easier entry counts: %d (%v)", n, err)
	}
	if n, _ := s.EasierRequestsSince(ctx, testUser, time.Now().Add(time.Hour)); n != 0 {
		t.Errorf("easier since the future = %d", n)
	}
}

// CreateThreadQuestion: the same root (lecture, quote, mode, question) or
// follow-up (parent, mode, question) within the window returns the first
// entry; another key, an expired window or a failed entry creates a new one.
func TestCreateThreadQuestionDedupe(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	lec, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, testUser))
	if err != nil {
		t.Fatal(err)
	}
	const window = 20 * time.Second
	rootQ := models.Question{LectureID: lec.ID, Anchor: "s1", Kind: "question", Question: "더 쉽게", Quote: "구절", Mode: learning.ThreadEasier}
	first, created, err := s.CreateThreadQuestion(ctx, testUser, rootQ, window)
	if err != nil || !created {
		t.Fatalf("first root: %+v created=%v (%v)", first, created, err)
	}
	again, created, err := s.CreateThreadQuestion(ctx, testUser, rootQ, window)
	if err != nil || created || again.ID != first.ID {
		t.Fatalf("double submit must return the first root: %+v created=%v (%v)", again, created, err)
	}
	deeper := rootQ
	deeper.Mode = learning.ThreadDeeper
	if q, created, _ := s.CreateThreadQuestion(ctx, testUser, deeper, window); !created || q.ID == first.ID {
		t.Fatal("another mode is a new root")
	}
	if q, created, _ := s.CreateThreadQuestion(ctx, "someone-else", rootQ, window); !created || q.ID == first.ID {
		t.Fatal("another user's submit is a new root")
	}
	fq := models.Question{LectureID: lec.ID, Kind: "question", Question: "왜?", ParentID: first.ID, Mode: learning.ThreadQuestion}
	f1, created, err := s.CreateThreadQuestion(ctx, testUser, fq, window)
	if err != nil || !created {
		t.Fatalf("first follow-up: %v", err)
	}
	if f2, created, _ := s.CreateThreadQuestion(ctx, testUser, fq, window); created || f2.ID != f1.ID {
		t.Fatal("double follow-up must return the first one")
	}
	if _, err := s.FailQuestion(ctx, f1.ID, "m", "x"); err != nil {
		t.Fatal(err)
	}
	if f3, created, _ := s.CreateThreadQuestion(ctx, testUser, fq, window); !created || f3.ID == f1.ID {
		t.Fatal("a failed follow-up must not swallow the retry")
	}
	if _, err := s.DB().ExecContext(ctx, `UPDATE lecture_questions SET created_at = ? WHERE id = ?`,
		FormatTime(time.Now().Add(-time.Minute)), first.ID); err != nil {
		t.Fatal(err)
	}
	if q, created, _ := s.CreateThreadQuestion(ctx, testUser, rootQ, window); !created || q.ID == first.ID {
		t.Fatal("after the window the same root is a new entry")
	}
}

// The level-change stamp moves only when the level changes; dismiss stores its time.
func TestLevelSuggestionSettingsColumns(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	if err := s.DismissLevelSuggestion(ctx, testUser, time.Now()); err != nil {
		t.Fatalf("dismiss without settings: %v", err)
	}
	level := learning.ReaderPractitioner
	st, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &level})
	if err != nil || st.ReaderLevelChangedAt == nil || st.LevelSuggestionDismissedAt != nil {
		t.Fatalf("first level save: %+v (%v)", st, err)
	}
	first := *st.ReaderLevelChangedAt
	time.Sleep(5 * time.Millisecond)
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &level}); err != nil || !st.ReaderLevelChangedAt.Equal(first) {
		t.Fatalf("same level must keep the stamp: %v (%v)", st.ReaderLevelChangedAt, err)
	}
	daily := 45
	if st, _ = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{DailyMinutes: &daily}); !st.ReaderLevelChangedAt.Equal(first) {
		t.Fatalf("another field must keep the stamp: %v", st.ReaderLevelChangedAt)
	}
	lower := learning.ReaderNovice
	if st, _ = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &lower}); !st.ReaderLevelChangedAt.After(first) {
		t.Fatalf("a new level must move the stamp: %v", st.ReaderLevelChangedAt)
	}
	at := time.Now().UTC().Truncate(time.Millisecond)
	if err := s.DismissLevelSuggestion(ctx, testUser, at); err != nil {
		t.Fatal(err)
	}
	if st, err = s.GetLearningSettings(ctx, testUser); err != nil || st.LevelSuggestionDismissedAt == nil || !st.LevelSuggestionDismissedAt.Equal(at) {
		t.Fatalf("dismissed at = %v, want %v (%v)", st.LevelSuggestionDismissedAt, at, err)
	}
}

// TestUpgradeFromV020ReadingThreads: 021 only adds columns — a 020 question
// reads as a question root without a quote, settings keep their level and
// have no stamps, and threads work right after the upgrade.
func TestUpgradeFromV020ReadingThreads(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "021")); err != nil {
		t.Fatalf("020 migrate: %v", err)
	}
	if _, err := old.DB().ExecContext(ctx, `INSERT INTO lecture_questions (id, user_id, anchor, kind, question, answer, status)
		VALUES ('old-q', ?, 's1', 'question', '옛 질문', '옛 답', 'answered')`, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := old.DB().ExecContext(ctx, `INSERT INTO user_learning_settings (user_id, example_language, reader_level) VALUES (?, 'typescript', 'advanced')`, testUser); err != nil {
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
	q, err := s.GetQuestion(ctx, testUser, "old-q")
	if err != nil || q.Mode != learning.ThreadQuestion || q.Quote != "" || q.ParentID != "" || q.Answer != "옛 답" {
		t.Fatalf("old question after upgrade: %+v (%v)", q, err)
	}
	st, err := s.GetLearningSettings(ctx, testUser)
	if err != nil || st.ReaderLevel != learning.ReaderAdvanced || st.ReaderLevelChangedAt != nil || st.LevelSuggestionDismissedAt != nil {
		t.Fatalf("settings after upgrade: %+v (%v)", st, err)
	}
	child, err := s.CreateQuestion(ctx, testUser, models.Question{Kind: "question", Question: "이어서", ParentID: "old-q"})
	if err != nil || child.ParentID != "old-q" {
		t.Fatalf("follow-up after upgrade: %+v (%v)", child, err)
	}
	if thread, err := s.ListThread(ctx, testUser, "old-q"); err != nil || len(thread) != 2 {
		t.Fatalf("thread after upgrade: %d (%v)", len(thread), err)
	}
}
