package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// conceptDismissedSQL: concept column ref (e.g. "c.id") is marked '필요 없음' by
// user (an SQL expression). Separate from user_concept_states (이미 앎/완료):
// it never unlocks dependents and never changes mastery.
func conceptDismissedSQL(conceptRef, user string) string {
	return "EXISTS (SELECT 1 FROM user_concept_dismissals dm WHERE dm.user_id = " + user +
		" AND dm.concept_id = " + conceptRef + ")"
}

// DismissConcept marks a visible, non-retired concept '필요 없음' for the user
// (idempotent). ErrNotFound when the user cannot see it, ErrConceptRetired when retired.
func (s *SQLiteStore) DismissConcept(ctx context.Context, userID, conceptID string) error {
	if err := s.requireLiveConcept(ctx, userID, conceptID); err != nil {
		return err
	}
	if _, err := s.c().exec(ctx, `
		INSERT INTO user_concept_dismissals (user_id, concept_id) VALUES ($1, $2)
		ON CONFLICT (user_id, concept_id) DO NOTHING`, userID, conceptID); err != nil {
		return fmt.Errorf("dismiss concept: %w", err)
	}
	return nil
}

// RestoreConcept undoes DismissConcept (idempotent): the concept returns to
// recommendations and its paused reviews resume with their stored dates.
// A retired concept can still be restored (nothing to resume, but no stale row).
func (s *SQLiteStore) RestoreConcept(ctx context.Context, userID, conceptID string) error {
	if err := s.requireLiveConcept(ctx, userID, conceptID); err != nil && !errors.Is(err, ErrConceptRetired) {
		return err
	}
	if _, err := s.c().exec(ctx, `
		DELETE FROM user_concept_dismissals WHERE user_id = $1 AND concept_id = $2`, userID, conceptID); err != nil {
		return fmt.Errorf("restore concept: %w", err)
	}
	return nil
}

// ConceptDismissals returns the concepts the user marked '필요 없음'.
// (DismissedConceptIDs is a different signal: concepts whose lectures were all hidden from 이어보기.)
func (s *SQLiteStore) ConceptDismissals(ctx context.Context, userID string) (map[string]bool, error) {
	ids, err := collect(s.c().query(ctx,
		`SELECT concept_id FROM user_concept_dismissals WHERE user_id = $1`, userID), scanString)
	if err != nil {
		return nil, fmt.Errorf("concept dismissals: %w", err)
	}
	out := make(map[string]bool, len(ids))
	for _, id := range ids {
		out[id] = true
	}
	return out, nil
}

func (s *SQLiteStore) requireLiveConcept(ctx context.Context, userID, conceptID string) error {
	var status string
	err := s.c().row(ctx, `SELECT c.status FROM concepts c WHERE c.id = $1 AND `+visibleTo("c", "$2"),
		conceptID, userID).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("load concept: %w", err)
	}
	if status == "retired" {
		return ErrConceptRetired
	}
	return nil
}
