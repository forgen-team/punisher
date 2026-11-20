package store

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/llm"
	"github.com/forgen-team/punisher/migrations"
)

// aiProvider defaults to claude, saves, keeps other fields and reads unknown
// stored values as claude; lectures and jobs record their provider.
func TestAIProviderSettingAndRecords(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	if st, err := s.GetLearningSettings(ctx, testUser); err != nil || st.AIProvider != llm.ProviderClaude {
		t.Fatalf("default provider = %q (%v)", st.AIProvider, err)
	}
	level := learning.ReaderAdvanced
	if _, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{ReaderLevel: &level}); err != nil {
		t.Fatal(err)
	}
	if got := s.AIProviderFor(ctx, testUser); got != llm.ProviderClaude {
		t.Fatalf("first save without aiProvider = %q", got)
	}
	codex := llm.ProviderCodex
	st, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{AIProvider: &codex})
	if err != nil || st.AIProvider != codex || st.ReaderLevel != level {
		t.Fatalf("save codex = %+v (%v)", st, err)
	}
	if got := s.AIProviderFor(ctx, testUser); got != codex {
		t.Fatalf("AIProviderFor = %q", got)
	}
	if _, err := s.DB().ExecContext(ctx, `UPDATE user_learning_settings SET ai_provider = 'gemini' WHERE user_id = ?`, testUser); err != nil {
		t.Fatal(err)
	}
	if got := s.AIProviderFor(ctx, testUser); got != llm.ProviderClaude {
		t.Fatalf("unknown stored provider must read as claude: %q", got)
	}

	conceptID, objectiveID := masteryFixture(t, s, testUser)
	old, err := s.InsertLecture(ctx, testLecture(conceptID, objectiveID, learning.LocaleKO, testUser))
	if err != nil || old.Provider != llm.ProviderClaude {
		t.Fatalf("lecture without provider = %q (%v)", old.Provider, err)
	}
	nl := testLecture(conceptID, objectiveID, learning.LocaleKO, testUser)
	nl.Provider = llm.ProviderAgy
	lec, err := s.InsertLecture(ctx, nl)
	if err != nil || lec.Provider != llm.ProviderAgy {
		t.Fatalf("agy lecture = %q (%v)", lec.Provider, err)
	}
	// The provider is not a reuse key: the newest lecture of the target is found whoever wrote it.
	if got, err := s.LatestLectureForTarget(ctx, learning.JobInstant, conceptID, objectiveID, 1, learning.LocaleKO, "", []string{"published"}); err != nil || got.ID != lec.ID {
		t.Fatalf("reuse ignores the provider: %v %v", got.ID, err)
	}

	if err := s.RecordUsage(ctx, testUser, "gloss", "", JobOutcome{Status: "succeeded", Provider: llm.ProviderCodex}); err != nil {
		t.Fatal(err)
	}
	job, _, err := s.CreateJob(ctx, NewJob{IdempotencyKey: "k1", Kind: learning.JobInstant, UserID: testUser, ConceptID: conceptID, ObjectiveKey: objectiveID, Depth: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.FinishJob(ctx, job.ID, JobOutcome{Status: "failed", Provider: llm.ProviderAgy}); err != nil {
		t.Fatal(err)
	}
	var usageProvider, jobProvider string
	if err := s.DB().QueryRowContext(ctx, `SELECT provider FROM generation_jobs WHERE kind = 'gloss'`).Scan(&usageProvider); err != nil {
		t.Fatal(err)
	}
	if err := s.DB().QueryRowContext(ctx, `SELECT provider FROM generation_jobs WHERE id = ?`, job.ID).Scan(&jobProvider); err != nil {
		t.Fatal(err)
	}
	if usageProvider != llm.ProviderCodex || jobProvider != llm.ProviderAgy {
		t.Fatalf("job providers = %q, %q", usageProvider, jobProvider)
	}
}

// TestUpgradeFromV021AIProvider: 022 only adds columns — saved settings,
// lectures and jobs from 021 read as claude.
func TestUpgradeFromV021AIProvider(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "022")); err != nil {
		t.Fatalf("021 migrate: %v", err)
	}
	if _, err := old.DB().ExecContext(ctx, `INSERT INTO user_learning_settings (user_id, example_language, reader_level) VALUES (?, 'go', 'novice')`, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := old.DB().ExecContext(ctx, `INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, status) VALUES ('old-job', 'old-key', 'gloss', ?, 'succeeded')`, testUser); err != nil {
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
	if err != nil || st.AIProvider != llm.ProviderClaude || st.ExampleLanguage != "go" || st.ReaderLevel != learning.ReaderNovice {
		t.Fatalf("settings after upgrade: %+v (%v)", st, err)
	}
	var provider string
	if err := s.DB().QueryRowContext(ctx, `SELECT provider FROM generation_jobs WHERE id = 'old-job'`).Scan(&provider); err != nil || provider != llm.ProviderClaude {
		t.Fatalf("old job provider = %q (%v)", provider, err)
	}
	codex := llm.ProviderCodex
	if st, err := s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{AIProvider: &codex}); err != nil || st.AIProvider != codex || st.ExampleLanguage != "go" {
		t.Fatalf("save after upgrade: %+v (%v)", st, err)
	}
}
