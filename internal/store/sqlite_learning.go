package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/forgen-team/punisher/internal/learning"
)

// UpdateProgress upserts the reading position (percent never decreases once
// completed) and moves the concept to in_progress unless it is already done.
func (s *SQLiteStore) UpdateProgress(ctx context.Context, userID, lectureID, anchor string, percent int) error {
	conceptID, err := s.lectureConcept(ctx, userID, lectureID)
	if err != nil {
		return err
	}
	if _, err := s.c().exec(ctx, `
		INSERT INTO user_lecture_progress (user_id, lecture_id, anchor, percent)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_id, lecture_id) DO UPDATE SET
			anchor = excluded.anchor,
			percent = CASE WHEN user_lecture_progress.completed_at IS NOT NULL
			               THEN MAX(user_lecture_progress.percent, excluded.percent)
			               ELSE excluded.percent END,
			last_seen_at = NOW(),
			dismissed_at = NULL`, userID, lectureID, anchor, percent); err != nil {
		return fmt.Errorf("upsert progress: %w", err)
	}
	_, err = s.SetConceptState(ctx, userID, conceptID, learning.InProgress)
	return err
}

func (s *SQLiteStore) lectureConcept(ctx context.Context, userID, lectureID string) (string, error) {
	var conceptID string
	err := s.c().row(ctx, `
		SELECT concept_id FROM lectures
		WHERE id = $1 AND (kind = 'library' OR requested_by = $2)`, lectureID, userID).Scan(&conceptID)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return conceptID, err
}

// CompleteResult reports what CompleteLecture changed.
type CompleteResult struct {
	ConceptID           string   `json:"conceptId"`
	ObjectivesCompleted []string `json:"objectivesCompleted"`
	ConceptState        string   `json:"conceptState"`
}

// CompleteLecture marks the lecture's objectives completed at the lecture
// depth (dedup key (user, concept, objective, depth)), closes progress, and
// sets the concept to completed when no active objective remains pending
// (known stays known). Idempotent.
func (s *SQLiteStore) CompleteLecture(ctx context.Context, userID, lectureID string) (CompleteResult, error) {
	var res CompleteResult
	var objectiveIDs []string
	err := s.tx(ctx, func(c conn) error {
		var depth int
		err := c.row(ctx, `
			SELECT concept_id, depth, objective_ids FROM lectures
			WHERE id = $1 AND (kind = 'library' OR requested_by = $2)`, lectureID, userID).
			Scan(&res.ConceptID, &depth, strs(&objectiveIDs))
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load lecture: %w", err)
		}

		if _, err := c.exec(ctx, `
			INSERT INTO user_objective_states (user_id, concept_id, objective_id, depth, state, lecture_id)
			SELECT $1, $2, o.id, $3, 'completed', $4
			FROM objectives o WHERE o.id IN (SELECT value FROM json_each($5))
			ON CONFLICT (user_id, concept_id, objective_id, depth) DO UPDATE SET
				state = 'completed', lecture_id = excluded.lecture_id, updated_at = NOW()
			WHERE user_objective_states.state = 'in_progress'`,
			userID, res.ConceptID, depth, lectureID, objectiveIDs); err != nil {
			return fmt.Errorf("complete objectives: %w", err)
		}
		if _, err := c.exec(ctx, `
			INSERT INTO user_lecture_progress (user_id, lecture_id, percent, completed_at)
			VALUES ($1, $2, 100, NOW())
			ON CONFLICT (user_id, lecture_id) DO UPDATE SET percent = 100,
				completed_at = COALESCE(user_lecture_progress.completed_at, NOW()), last_seen_at = NOW()`,
			userID, lectureID); err != nil {
			return fmt.Errorf("complete progress: %w", err)
		}

		var pending bool
		if err := c.row(ctx, pendingObjectivesSQL, userID, res.ConceptID).Scan(&pending); err != nil {
			return fmt.Errorf("check pending: %w", err)
		}
		next := learning.Completed
		if pending {
			next = learning.InProgress
		}
		if err := c.row(ctx, `
			INSERT INTO user_concept_states (user_id, concept_id, state, completed_at)
			VALUES ($1, $2, $3, CASE WHEN $3 = 'completed' THEN NOW() END)
			ON CONFLICT (user_id, concept_id) DO UPDATE SET
				state = CASE WHEN user_concept_states.state IN ('known', 'completed') THEN user_concept_states.state
				             ELSE excluded.state END,
				completed_at = COALESCE(user_concept_states.completed_at, excluded.completed_at),
				updated_at = NOW()
			RETURNING state`, userID, res.ConceptID, next).Scan(&res.ConceptState); err != nil {
			return fmt.Errorf("update concept state: %w", err)
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	res.ObjectivesCompleted = objectiveIDs
	return res, nil
}

// pendingObjectivesSQL: does the concept still have an active objective that
// is not completed/known at its own depth? ($1 user, $2 concept)
const pendingObjectivesSQL = `
	SELECT EXISTS (
		SELECT 1 FROM objectives o
		WHERE o.concept_id = $2 AND o.status = 'active'
		  AND NOT EXISTS (
			SELECT 1 FROM user_objective_states us
			WHERE us.user_id = $1 AND us.objective_id = o.id AND us.depth = o.depth
			  AND us.state IN ('completed', 'known')))`

// InsertFeedback stores lecture feedback; altRequestID may be "".
func (s *SQLiteStore) InsertFeedback(ctx context.Context, userID, lectureID, feedback, anchor, comment, altRequestID string) (string, error) {
	var alt any
	if altRequestID != "" {
		alt = altRequestID
	}
	id := NewID()
	if _, err := s.c().exec(ctx, `
		INSERT INTO lecture_feedback (id, user_id, lecture_id, feedback, anchor, comment, alt_request_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, userID, lectureID, feedback, anchor, comment, alt); err != nil {
		return "", fmt.Errorf("insert feedback: %w", err)
	}
	return id, nil
}
