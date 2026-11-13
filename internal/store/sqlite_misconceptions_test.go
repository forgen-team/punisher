package store

import (
	"context"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/migrations"
)

// TestMigration013Misconceptions: 013 is recorded and creates user_misconceptions
// with the (user_id, turn_id) unique key and no CHECK on status (Go validates it).
func TestMigration013Misconceptions(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	var applied bool
	if err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = '013_misconceptions.sql')`).Scan(&applied); err != nil || !applied {
		t.Fatalf("013 applied = %v (%v)", applied, err)
	}
	var cols int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM pragma_table_info('user_misconceptions')
		WHERE name IN ('id', 'user_id', 'concept_id', 'objective_id', 'text', 'turn_id', 'status', 'created_at', 'resolved_at')`).Scan(&cols); err != nil || cols != 9 {
		t.Fatalf("columns = %d (%v)", cols, err)
	}
	var def string
	if err := s.c().row(ctx, `SELECT COALESCE(dflt_value, '') FROM pragma_table_info('user_misconceptions') WHERE name = 'status'`).Scan(&def); err != nil || def != "'open'" {
		t.Fatalf("status default = %q (%v)", def, err)
	}
	var unique int
	if err := s.c().row(ctx, `SELECT "unique" FROM pragma_index_list('user_misconceptions') WHERE name = 'user_misconceptions_turn'`).Scan(&unique); err != nil || unique != 1 {
		t.Fatalf("user_misconceptions_turn unique = %d (%v)", unique, err)
	}
	var keys string
	if err := s.c().row(ctx, `SELECT group_concat(name, ',') FROM (SELECT name FROM pragma_index_info('user_misconceptions_turn') ORDER BY seqno)`).Scan(&keys); err != nil || keys != "user_id,turn_id" {
		t.Fatalf("unique key = %q (%v)", keys, err)
	}
	var ddl string
	if err := s.c().row(ctx, `SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'user_misconceptions'`).Scan(&ddl); err != nil {
		t.Fatal(err)
	}
	if regexp.MustCompile(`(?i)\bCHECK\s*\(`).MatchString(ddl) {
		t.Fatalf("no CHECK constraint expected:\n%s", ddl)
	}
}

// judgedTurn inserts a check session, an item on objectiveID and a judged
// (ready) tutor turn whose grade carries misconception ("" = none).
func judgedTurn(t *testing.T, s *SQLiteStore, user, conceptID, objectiveID, misconception string) string {
	t.Helper()
	ctx := context.Background()
	sessionID, itemID, turnID := NewID(), NewID(), NewID()
	if _, err := s.c().exec(ctx, `INSERT INTO check_sessions (id, user_id, kind, concept_id, session_key, status)
		VALUES ($1, $2, 'check', $3, $4, 'finished')`, sessionID, user, conceptID, "test:"+sessionID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt, objective_text, objective_depth)
		VALUES ($1, $2, $3, 'check', 'recall', '왜?', '읽기 경로', 1)`, itemID, conceptID, objectiveID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.c().exec(ctx, `INSERT INTO check_turns (id, session_id, user_id, item_id, seq, role, move, verdict, outcome, grade)
		VALUES ($1, $2, $3, $4, 1, 'tutor', 'explain', 'incorrect', 'failed', json_object('misconception', $5))`,
		turnID, sessionID, user, itemID, misconception); err != nil {
		t.Fatal(err)
	}
	return turnID
}

// TestUpgradeFromV012 builds a database at 012 (the P1–P3 mastery branch: a
// judged turn with a misconception and an objective lapsed twice), then
// applies 013: the old turn can be acknowledged once, it resolves inside a
// transaction, and the lapsed objective shows up for the recheck card.
func TestUpgradeFromV012(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "013")); err != nil {
		t.Fatalf("012 migrate: %v", err)
	}
	conceptID, objectiveID := masteryFixture(t, old, testUser)
	turnID := judgedTurn(t, old, testUser, conceptID, objectiveID, "  두 번째 읽기도 디스크에서 읽는다 ")
	if _, err := old.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, lapses, next_review_at)
		VALUES ($1, $2, $3, 1, 'aided', 2, $4)`, testUser, objectiveID, conceptID, time.Now().Add(-time.Hour)); err != nil {
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
	m, created, err := s.AcknowledgeMisconception(ctx, testUser, turnID)
	if err != nil || !created || m.Text != "두 번째 읽기도 디스크에서 읽는다" || m.ObjectiveID != objectiveID ||
		m.ConceptName != "캐시" || m.ObjectiveText != "읽기 경로" || m.Status != learning.MisconceptionOpen || m.CreatedAt.IsZero() || m.ResolvedAt != nil {
		t.Fatalf("acknowledged after 013 = %+v created=%v (%v)", m, created, err)
	}
	if again, created, err := s.AcknowledgeMisconception(ctx, testUser, turnID); err != nil || created || again.ID != m.ID {
		t.Fatalf("second acknowledgement = %+v created=%v (%v)", again, created, err)
	}
	var rows int
	if err := s.c().row(ctx, `SELECT COUNT(*) FROM user_misconceptions WHERE user_id = $1`, testUser).Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("one row per turn: %d (%v)", rows, err)
	}
	if open, err := s.OpenMisconceptions(ctx, testUser, conceptID, 3); err != nil || len(open) != 1 {
		t.Fatalf("open = %+v (%v)", open, err)
	}

	d, err := s.LoadToday(ctx, testUser)
	if err != nil {
		t.Fatal(err)
	}
	if lapsed, ok := d.Lapsed[conceptID]; !ok || !lapsed.Due || lapsed.LectureID != "" {
		t.Fatalf("lapsed = %+v", d.Lapsed)
	}

	if err := s.tx(ctx, func(c conn) error { return resolveMisconceptions(ctx, c, testUser, objectiveID) }); err != nil {
		t.Fatal(err)
	}
	resolved, err := s.ListMisconceptions(ctx, testUser, learning.MisconceptionResolved, 10)
	if err != nil || len(resolved) != 1 || resolved[0].ResolvedAt == nil || resolved[0].ResolvedAt.Before(m.CreatedAt) {
		t.Fatalf("resolved = %+v (%v)", resolved, err)
	}
	if open, _ := s.ListMisconceptions(ctx, testUser, learning.MisconceptionOpen, 10); len(open) != 0 {
		t.Fatalf("nothing open: %+v", open)
	}
}

// TestMisconceptionSQLiteSemantics: a turn without a misconception (or another
// user's turn) is not found, the same timestamp orders by rowid (latest first),
// and lapses below learning.RecheckLapses or a queued objective are not due.
func TestMisconceptionSQLiteSemantics(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	if _, _, err := s.AcknowledgeMisconception(ctx, testUser, judgedTurn(t, s, testUser, conceptID, objectiveID, "   ")); err != ErrNotFound {
		t.Fatalf("blank misconception → not found: %v", err)
	}
	first := judgedTurn(t, s, testUser, conceptID, objectiveID, "첫째")
	if _, _, err := s.AcknowledgeMisconception(ctx, "test-other-user", first); err != ErrNotFound {
		t.Fatalf("another user's turn → not found: %v", err)
	}
	second := judgedTurn(t, s, testUser, conceptID, objectiveID, "둘째")
	for _, id := range []string{first, second} {
		if _, _, err := s.AcknowledgeMisconception(ctx, testUser, id); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.c().exec(ctx, `UPDATE user_misconceptions SET created_at = '2026-11-13T00:00:00.000Z'`); err != nil {
		t.Fatal(err)
	}
	list, err := s.ListMisconceptions(ctx, testUser, learning.MisconceptionOpen, 10)
	if err != nil || len(list) != 2 || list[0].Text != "둘째" || list[1].Text != "첫째" {
		t.Fatalf("latest first = %+v (%v)", list, err)
	}

	if _, err := s.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, best, lapses, next_review_at, queued)
		VALUES ($1, $2, $3, 1, 'aided', 1, $4, 0)`, testUser, objectiveID, conceptID, time.Now().Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if d, err := s.LoadToday(ctx, testUser); err != nil || len(d.Lapsed) != 0 {
		t.Fatalf("one lapse → none: %+v (%v)", d.Lapsed, err)
	}
	if _, err := s.c().exec(ctx, `UPDATE user_objective_mastery SET lapses = 2, queued = 1 WHERE objective_id = $1`, objectiveID); err != nil {
		t.Fatal(err)
	}
	d, err := s.LoadToday(ctx, testUser)
	if lapsed, ok := d.Lapsed[conceptID]; err != nil || !ok || lapsed.Due {
		t.Fatalf("queued → listed but not due: %+v (%v)", d.Lapsed, err)
	}
}
