package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	seedfiles "github.com/forgen-team/punisher/curriculum"
	"github.com/forgen-team/punisher/internal/curriculum"
	"github.com/forgen-team/punisher/internal/learning"
	"github.com/forgen-team/punisher/internal/models"
	"github.com/forgen-team/punisher/migrations"
)

// prereqPair returns a seed prerequisite edge (from → to) whose `from` has no
// prerequisite itself (a frontier concept for a fresh user).
func prereqPair(t *testing.T, s *SQLiteStore) (from, to string) {
	t.Helper()
	edges, err := s.PrerequisiteEdges(context.Background(), testUser)
	if err != nil {
		t.Fatal(err)
	}
	hasPrereq := map[string]bool{}
	for _, e := range edges {
		hasPrereq[e.To] = true
	}
	for _, e := range edges {
		if !hasPrereq[e.From] {
			return e.From, e.To
		}
	}
	t.Fatal("no root prerequisite edge in the seed")
	return "", ""
}

// TestConceptDismissAndRestore: '필요 없음' is a flag beside the state — the
// concept keeps not_started, its dependents stay locked (and show the dismissed
// prerequisite), today's input carries it, and undo clears it. Idempotent both ways.
func TestConceptDismissAndRestore(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	from, to := prereqPair(t, s)

	for range 2 {
		if err := s.DismissConcept(ctx, testUser, from); err != nil {
			t.Fatalf("dismiss: %v", err)
		}
	}
	d, err := s.GetConceptDetail(ctx, testUser, from)
	if err != nil || !d.Dismissed || d.UserState != learning.NotStarted {
		t.Fatalf("dismissed detail: dismissed=%v state=%q err=%v", d.Dismissed, d.UserState, err)
	}
	dep, err := s.GetConceptDetail(ctx, testUser, to)
	if err != nil {
		t.Fatal(err)
	}
	if !dep.Locked {
		t.Error("a dismissed prerequisite must still lock its dependent")
	}
	if !prereqDismissed(dep.Prerequisites, from) {
		t.Errorf("dependent's prerequisite %s must be marked dismissed: %+v", from, dep.Prerequisites)
	}
	today, err := s.LoadToday(ctx, testUser, time.Now())
	if err != nil || !today.Concepts[from].Dismissed || today.Concepts[to].Dismissed {
		t.Fatalf("today dismissed flags: %v / %v (%v)", today.Concepts[from].Dismissed, today.Concepts[to].Dismissed, err)
	}
	if got, err := s.ConceptDismissals(ctx, testUser); err != nil || !got[from] || len(got) != 1 {
		t.Fatalf("dismissals = %v (%v)", got, err)
	}
	if got, _ := s.ConceptDismissals(ctx, "someone-else"); len(got) != 0 {
		t.Errorf("dismissal leaked to another user: %v", got)
	}

	for range 2 {
		if err := s.RestoreConcept(ctx, testUser, from); err != nil {
			t.Fatalf("restore: %v", err)
		}
	}
	if d, _ := s.GetConceptDetail(ctx, testUser, from); d.Dismissed {
		t.Error("restore must clear dismissed")
	}
	if err := s.DismissConcept(ctx, testUser, "no.such-concept"); !errors.Is(err, ErrNotFound) {
		t.Errorf("unknown concept: %v, want ErrNotFound", err)
	}
}

func prereqDismissed(links []models.ConceptLink, id string) bool {
	for _, l := range links {
		if l.ID == id {
			return l.Dismissed
		}
	}
	return false
}

// TestDismissPausesReviews: a dismissed concept's objectives leave the review
// queue and counts (the mastery row is kept); undo brings them back.
func TestDismissPausesReviews(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	conceptID, objectiveID := masteryFixture(t, s, testUser)
	now := time.Now().UTC()
	if _, err := s.c().exec(ctx, `INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, grad_step, next_review_at, queued)
		VALUES ($1, $2, $3, 1, 0, $4, 0)`, testUser, objectiveID, conceptID, now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	due := func() int {
		t.Helper()
		q, err := s.ReviewQueue(ctx, testUser, now)
		if err != nil {
			t.Fatal(err)
		}
		n, err := s.ReviewCounts(ctx, testUser, now)
		if err != nil || n.Due != len(q) {
			t.Fatalf("counts %+v vs queue %d (%v)", n, len(q), err)
		}
		return len(q)
	}
	if due() != 1 {
		t.Fatal("fixture review must be due")
	}
	if err := s.DismissConcept(ctx, testUser, conceptID); err != nil {
		t.Fatal(err)
	}
	if n := due(); n != 0 {
		t.Errorf("dismissed concept still has %d due reviews", n)
	}
	if err := s.RestoreConcept(ctx, testUser, conceptID); err != nil {
		t.Fatal(err)
	}
	if due() != 1 {
		t.Error("undo must resume the paused review")
	}
}

// TestUpgradeFromV019ConceptDismissals: 020 only adds the dismissal table — a
// 019 database keeps its states and can dismiss right after the upgrade.
func TestUpgradeFromV019ConceptDismissals(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "punisher.db")
	old, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := old.Migrate(ctx, migrationsBefore(t, "020")); err != nil {
		t.Fatalf("019 migrate: %v", err)
	}
	seed, err := curriculum.Load(seedfiles.Files)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.SyncCurriculum(ctx, seed); err != nil {
		t.Fatal(err)
	}
	from, to := prereqPair(t, old)
	if _, err := old.SetConceptState(ctx, testUser, from, learning.Known); err != nil {
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
	if err := s.DismissConcept(ctx, testUser, to); err != nil {
		t.Fatalf("dismiss after upgrade: %v", err)
	}
	known, err := s.GetConceptDetail(ctx, testUser, from)
	if err != nil || known.UserState != learning.Known || known.Dismissed {
		t.Fatalf("known state after upgrade: %q dismissed=%v (%v)", known.UserState, known.Dismissed, err)
	}
	if d, err := s.GetConceptDetail(ctx, testUser, to); err != nil || !d.Dismissed {
		t.Fatalf("dismissed after upgrade: %v (%v)", d.Dismissed, err)
	}
}
