package store

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/migrations"
)

// TestMigration012Pretest: 012 is recorded and adds only the two columns, with
// defaults (SQLite: INTEGER 1 / 0 for witems' BOOLEAN TRUE / FALSE).
func TestMigration012Pretest(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	var applied bool
	if err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '012_pretest.sql')`).Scan(&applied); err != nil || !applied {
		t.Fatalf("012 applied = %v (%v)", applied, err)
	}
	cols := map[string]string{"user_learning_settings": "pretest_enabled", "user_objective_mastery": "verify_queued"}
	want := map[string]string{"pretest_enabled": "1", "verify_queued": "0"}
	for table, col := range cols {
		var typ, def string
		var notNull int
		if err := s.c().row(ctx, `SELECT type, COALESCE(dflt_value, ''), "notnull" FROM pragma_table_info($1) WHERE name = $2`,
			table, col).Scan(&typ, &def, &notNull); err != nil {
			t.Fatalf("%s.%s: %v", table, col, err)
		}
		if typ != "INTEGER" || def != want[col] || notNull != 1 {
			t.Fatalf("%s.%s = %s default %q notnull %d", table, col, typ, def, notNull)
		}
	}
	if st, err := s.GetLearningSettings(ctx, "test-012-nobody"); err != nil || !st.PretestEnabled {
		t.Fatalf("default settings: %+v %v", st, err)
	}
}

// TestUpgradeFromV011 builds a database at 011 (the P1·P2 mastery branch:
// saved settings and evidence), then applies 012: the settings get
// pretest_enabled on and can turn it off, existing evidence is not queued.
func TestUpgradeFromV011(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "012")); err != nil {
		t.Fatalf("011 migrate: %v", err)
	}
	conceptID, objectiveID := masteryFixture(t, old, testUser)
	if _, err := old.c().exec(ctx, `INSERT INTO user_learning_settings (user_id, example_language, daily_minutes, review_minutes)
		VALUES ($1, 'go', 45, 20)`, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := old.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best)
		VALUES ($1, $2, $3, 2, 'unaided')`, testUser, objectiveID, conceptID); err != nil {
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
	if err != nil || !st.PretestEnabled || st.ReviewMinutes != 20 || st.DailyMinutes != 45 {
		t.Fatalf("settings after 012 = %+v (%v)", st, err)
	}
	off := false
	if st, err = s.UpdateLearningSettings(ctx, testUser, LearningSettingsPatch{PretestEnabled: &off}); err != nil || st.PretestEnabled || st.ReviewMinutes != 20 {
		t.Fatalf("patch after 012 = %+v (%v)", st, err)
	}
	var m learning.Mastery
	if err := s.tx(ctx, func(c conn) error {
		l, err := lockMastery(ctx, c, testUser, objectiveID)
		m = l.M
		return err
	}); err != nil || m.Box != 2 || m.Verify {
		t.Fatalf("evidence after 012 = %+v (%v)", m, err)
	}
	if n, err := s.QueueKnownVerify(ctx, testUser, conceptID); err != nil || n != 0 {
		t.Fatalf("an objective with evidence is not queued: %d %v", n, err)
	}
}

// TestKnownVerifySQLiteSemantics: the upsert queues only objectives without
// evidence and only once (RowsAffected), and the targets are grouped per
// concept in queue order with depth/ordinal order inside (array_agg → Go grouping).
func TestKnownVerifySQLiteSemantics(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	first, firstObj := masteryFixture(t, s, testUser)
	if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, '쓰기 경로', 2, 2)`,
		first+"#2", first); err != nil {
		t.Fatal(err)
	}
	second := "test.mastery-second"
	if _, err := s.c().exec(ctx, `INSERT INTO concepts (id, name, area, depth, owner_user_id) VALUES ($1, '로그', 'os', 1, $2)`, second, testUser); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO objectives (id, concept_id, text, depth, ordinal) VALUES ($1, $2, '순서', 1, 1)`, second+"#1", second); err != nil {
		t.Fatal(err)
	}
	for _, c := range []string{first, second} {
		if _, err := s.c().exec(ctx, `INSERT INTO user_concept_states (user_id, concept_id, state) VALUES ($1, $2, 'known')`, testUser, c); err != nil {
			t.Fatal(err)
		}
	}
	if n, err := s.QueueKnownVerify(ctx, testUser, first); err != nil || n != 2 {
		t.Fatalf("queue first = %d %v", n, err)
	}
	if n, err := s.QueueKnownVerify(ctx, testUser, first); err != nil || n != 0 {
		t.Fatalf("queued once = %d %v", n, err)
	}
	if n, err := s.QueueKnownVerify(ctx, testUser, second); err != nil || n != 1 {
		t.Fatalf("queue second = %d %v", n, err)
	}
	all, err := s.KnownVerifyTargets(ctx, testUser, 5)
	if err != nil || len(all) != 2 || all[0].ConceptID != first || len(all[0].ObjectiveIDs) != 2 ||
		all[0].ObjectiveIDs[0] != firstObj || all[1].ConceptID != second {
		t.Fatalf("targets = %+v %v", all, err)
	}
	if one, err := s.KnownVerifyTargets(ctx, testUser, 1); err != nil || len(one) != 1 || len(one[0].ObjectiveIDs) != 2 {
		t.Fatalf("limit counts concepts: %+v %v", one, err)
	}
	if n, err := s.KnownVerifyReady(ctx, testUser, time.Now()); err != nil || n != 0 {
		t.Fatalf("no items yet: %d %v", n, err)
	}
}
