package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
)

// VariantTarget is a soon-due objective without an unseen item: a review
// variant is generated from its latest lecture (design §5 P2).
type VariantTarget struct {
	ObjectiveID string
	ConceptID   string
	LectureID   string
	// Seen: prompts of the objective's visible items (the variant must differ).
	Seen []string
}

// ReviewVariantTargets lists objectives due before `until` (in the queue: not
// queued, not graduated, active) whose visible items were all asked already,
// with the newest lecture that has an item for them. Earliest due first.
// JOIN LATERAL (… LIMIT 1) src → a correlated scalar subquery, kept only when present.
func (s *SQLiteStore) ReviewVariantTargets(ctx context.Context, userID string, until time.Time, limit int) ([]VariantTarget, error) {
	targets, err := collect(s.c().query(ctx, `
		SELECT objective_id, concept_id, lecture_id FROM (
			SELECT m.objective_id, m.concept_id, m.next_review_at,
				(SELECT ci.lecture_id FROM check_items ci
				 WHERE ci.objective_id = m.objective_id AND ci.lecture_id IS NOT NULL AND ci.status = 'active' AND `+checkItemVisible+`
				 ORDER BY ci.created_at DESC, ci.rowid DESC LIMIT 1) AS lecture_id
			FROM user_objective_mastery m`+reviewScope+`
			WHERE m.user_id = $1 AND NOT m.queued AND m.grad_step < 2 AND m.next_review_at < $2
			  AND NOT EXISTS (
				SELECT 1 FROM check_items ci
				WHERE ci.objective_id = m.objective_id AND ci.status = 'active' AND ci.purpose IN ('check', 'review')
				  AND `+checkItemVisible+`
				  AND NOT EXISTS (SELECT 1 FROM check_turns t WHERE t.user_id = $1 AND t.item_id = ci.id)))
		WHERE lecture_id IS NOT NULL
		ORDER BY next_review_at, objective_id LIMIT $3`, userID, until, limit), func(row rowScanner) (VariantTarget, error) {
		var v VariantTarget
		err := row.Scan(&v.ObjectiveID, &v.ConceptID, &v.LectureID)
		return v, err
	})
	if err != nil {
		return nil, fmt.Errorf("review variant targets: %w", err)
	}
	for i := range targets {
		seen, err := collect(s.c().query(ctx, `SELECT ci.prompt FROM check_items ci
			WHERE ci.objective_id = $2 AND `+checkItemVisible+` ORDER BY ci.created_at, ci.rowid`, userID, targets[i].ObjectiveID), scanString)
		if err != nil {
			return nil, fmt.Errorf("seen prompts: %w", err)
		}
		targets[i].Seen = seen
	}
	return targets, nil
}

// SaveVariantItems stores validated review variants under the lecture's key
// without retiring anything; each points to its objective's active check item
// of that lecture (variant_of).
func (s *SQLiteStore) SaveVariantItems(ctx context.Context, key CheckItemKey, conceptID, jobID, model, promptVersion string, items []check.Item) ([]CheckItem, error) {
	out := make([]CheckItem, 0, len(items))
	err := s.tx(ctx, func(c conn) error {
		for _, it := range items {
			var variantOf string
			err := c.row(ctx, `SELECT id FROM check_items WHERE lecture_id = $1 AND objective_id = $2
				AND purpose = 'check' AND status = 'active' ORDER BY created_at DESC, rowid DESC LIMIT 1`, key.LectureID, it.ObjectiveID).Scan(&variantOf)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("variant source: %w", err)
			}
			saved, err := insertCheckItem(ctx, c, key, conceptID, jobID, model, promptVersion, it, variantOf)
			if err != nil {
				return err
			}
			out = append(out, saved)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
