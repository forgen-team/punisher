package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/wooo-jin/punisher/internal/learning"
)

// SetConceptState applies an explicit state request (learning.NextState rules)
// in one transaction and returns the resulting state.
//
// completed/known also mark the concept's active objectives at their depth in
// user_objective_states — the dedup key (user, concept, objective, depth) —
// without overwriting an existing completed/known objective. A reset to
// not_started removes the concept row and only the objective rows that were
// created by concept-level marking (lecture_id IS NULL).
func (s *SQLiteStore) SetConceptState(ctx context.Context, userID, conceptID, requested string) (string, error) {
	var next string
	err := s.tx(ctx, func(c conn) error {
		var status string
		err := c.row(ctx, `SELECT c.status FROM concepts c WHERE c.id = $1 AND `+visibleTo("c", "$2"), conceptID, userID).Scan(&status)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load concept: %w", err)
		}
		if status == "retired" {
			return ErrConceptRetired
		}

		var current string
		err = c.row(ctx, `
			SELECT state FROM user_concept_states
			WHERE user_id = $1 AND concept_id = $2`, userID, conceptID).Scan(&current)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("load concept state: %w", err)
		}

		var changed bool
		next, changed, err = learning.NextState(current, requested)
		if err != nil || !changed {
			return err
		}
		return applyConceptState(ctx, c, userID, conceptID, next)
	})
	if err != nil {
		return "", err
	}
	return next, nil
}

func applyConceptState(ctx context.Context, c conn, userID, conceptID, next string) error {
	if next == learning.NotStarted {
		if _, err := c.exec(ctx,
			`DELETE FROM user_concept_states WHERE user_id = $1 AND concept_id = $2`, userID, conceptID); err != nil {
			return fmt.Errorf("reset concept state: %w", err)
		}
		if _, err := c.exec(ctx, `
			DELETE FROM user_objective_states
			WHERE user_id = $1 AND concept_id = $2 AND lecture_id IS NULL`, userID, conceptID); err != nil {
			return fmt.Errorf("reset objective states: %w", err)
		}
		return nil
	}

	if _, err := c.exec(ctx, `
		INSERT INTO user_concept_states (user_id, concept_id, state, completed_at, updated_at)
		VALUES ($1, $2, $3, CASE WHEN $3 = 'completed' THEN NOW() END, NOW())
		ON CONFLICT (user_id, concept_id) DO UPDATE SET
			state = excluded.state,
			completed_at = COALESCE(excluded.completed_at, user_concept_states.completed_at),
			updated_at = NOW()`, userID, conceptID, next); err != nil {
		return fmt.Errorf("upsert concept state: %w", err)
	}

	if !learning.IsDone(next) {
		return nil
	}
	if _, err := c.exec(ctx, `
		INSERT INTO user_objective_states (user_id, concept_id, objective_id, depth, state)
		SELECT $1, o.concept_id, o.id, o.depth, $3
		FROM objectives o
		WHERE o.concept_id = $2 AND o.status = 'active'
		ON CONFLICT (user_id, concept_id, objective_id, depth) DO UPDATE SET
			state = excluded.state, updated_at = NOW()
		WHERE user_objective_states.state = 'in_progress'`, userID, conceptID, next); err != nil {
		return fmt.Errorf("mark objectives: %w", err)
	}
	return nil
}
