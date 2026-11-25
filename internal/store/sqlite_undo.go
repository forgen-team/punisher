package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/forgen-team/punisher/internal/learning"
)

// Undo of the reversible learning actions ('되돌리기' toasts in the UI): lecture
// completion, '이미 앎' and abandoning a check. No new columns — each undo works
// from what the forward action wrote.

// UncompleteLecture undoes CompleteLecture: the lecture's objectives it marked
// completed (lecture_id = this lecture, the lecture's depth) go back to
// in_progress, the progress row loses completed_at, and a concept that is
// completed only because of them returns to in_progress (known stays known).
func (s *SQLiteStore) UncompleteLecture(ctx context.Context, userID, lectureID string) (CompleteResult, error) {
	var res CompleteResult
	err := s.tx(ctx, func(c conn) error {
		var depth int
		err := c.row(ctx, `
			SELECT concept_id, depth FROM lectures
			WHERE id = $1 AND (kind = 'library' OR requested_by = $2)`, lectureID, userID).Scan(&res.ConceptID, &depth)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load lecture: %w", err)
		}
		if _, err := c.exec(ctx, `
			UPDATE user_objective_states SET state = 'in_progress', updated_at = NOW()
			WHERE user_id = $1 AND concept_id = $2 AND lecture_id = $3 AND depth = $4 AND state = 'completed'`,
			userID, res.ConceptID, lectureID, depth); err != nil {
			return fmt.Errorf("reopen objectives: %w", err)
		}
		if _, err := c.exec(ctx, `
			UPDATE user_lecture_progress SET completed_at = NULL, last_seen_at = NOW()
			WHERE user_id = $1 AND lecture_id = $2`, userID, lectureID); err != nil {
			return fmt.Errorf("reopen progress: %w", err)
		}
		var pending bool
		if err := c.row(ctx, pendingObjectivesSQL, userID, res.ConceptID).Scan(&pending); err != nil {
			return fmt.Errorf("check pending: %w", err)
		}
		if pending {
			if _, err := c.exec(ctx, `
				UPDATE user_concept_states SET state = 'in_progress', completed_at = NULL, updated_at = NOW()
				WHERE user_id = $1 AND concept_id = $2 AND state = 'completed'`, userID, res.ConceptID); err != nil {
				return fmt.Errorf("reopen concept: %w", err)
			}
		}
		err = c.row(ctx, `SELECT state FROM user_concept_states WHERE user_id = $1 AND concept_id = $2`,
			userID, res.ConceptID).Scan(&res.ConceptState)
		if errors.Is(err, sql.ErrNoRows) {
			res.ConceptState = learning.NotStarted
			return nil
		}
		return err
	})
	res.ObjectivesCompleted = []string{}
	return res, err
}

// UndoKnown undoes '이미 앎' (SetConceptState known) back to previous, the state
// the concept had before (the UI sends it). Only a concept still marked known
// changes: objectives the marking created (lecture_id IS NULL, known) are
// removed, objectives it lifted from in_progress go back, and the concept row
// gets the previous state (not_started → no row). Queued verifications drop
// out by themselves (verifyScope needs the known state). Returns the new state.
func (s *SQLiteStore) UndoKnown(ctx context.Context, userID, conceptID, previous string) (string, error) {
	previous = learning.NormalizeState(previous)
	if !learning.IsValidState(previous) || previous == learning.Known {
		return "", fmt.Errorf("%w: %q", learning.ErrInvalidState, previous)
	}
	next := previous
	err := s.tx(ctx, func(c conn) error {
		var current string
		err := c.row(ctx, `SELECT state FROM user_concept_states WHERE user_id = $1 AND concept_id = $2`,
			userID, conceptID).Scan(&current)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && current != learning.Known) {
			next = learning.NormalizeState(current)
			return nil
		}
		if err != nil {
			return fmt.Errorf("load concept state: %w", err)
		}
		if _, err := c.exec(ctx, `
			DELETE FROM user_objective_states
			WHERE user_id = $1 AND concept_id = $2 AND state = 'known' AND lecture_id IS NULL`, userID, conceptID); err != nil {
			return fmt.Errorf("drop known objectives: %w", err)
		}
		if _, err := c.exec(ctx, `
			UPDATE user_objective_states SET state = 'in_progress', updated_at = NOW()
			WHERE user_id = $1 AND concept_id = $2 AND state = 'known'`, userID, conceptID); err != nil {
			return fmt.Errorf("reopen known objectives: %w", err)
		}
		if previous == learning.NotStarted {
			_, err := c.exec(ctx, `DELETE FROM user_concept_states WHERE user_id = $1 AND concept_id = $2`, userID, conceptID)
			return err
		}
		_, err = c.exec(ctx, `
			UPDATE user_concept_states SET state = $3, updated_at = NOW(),
				completed_at = CASE WHEN $3 = 'completed' THEN completed_at END
			WHERE user_id = $1 AND concept_id = $2`, userID, conceptID, previous)
		return err
	})
	if err != nil {
		return "", err
	}
	return next, nil
}

// ResumeCheckSession undoes a plain '그만하기': an abandoned session goes back
// to active when it was abandoned after since, still has an item to answer, and
// its current item was not closed by the abandon (result.items has no entry for
// the cursor — abandoning a half-answered item writes one, that one stays
// closed). ErrCheckClosed when not resumable, ErrCheckBusy when another open
// session of the same key exists.
func (s *SQLiteStore) ResumeCheckSession(ctx context.Context, userID, id string, since time.Time) error {
	res, err := s.c().exec(ctx, `
		UPDATE check_sessions SET status = 'active', finished_at = NULL, error = '', updated_at = NOW()
		WHERE user_id = $1 AND id = $2 AND status = 'abandoned' AND finished_at >= $3
		  AND json_array_length(item_ids) > cursor
		  AND COALESCE(json_array_length(json_extract(result, '$.items')), 0) <= cursor`,
		userID, id, FormatTime(since))
	if err != nil {
		if cerr := constraintErr(err); errors.Is(cerr, ErrCheckBusy) {
			return ErrCheckBusy
		}
		return fmt.Errorf("resume check session: %w", err)
	}
	if affected(res) == 0 {
		return ErrCheckClosed
	}
	return nil
}
