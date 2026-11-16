package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

var (
	clockMu   sync.Mutex
	clockLast time.Time
)

// clockNow replaces PostgreSQL's clock_timestamp() for user_objective_mastery.updated_at,
// the dispute's conflict marker (design §4.2): every call returns a distinct,
// increasing time in the stored millisecond precision, so two writes of one
// objective never share an updated_at even within the same millisecond.
func clockNow() time.Time {
	clockMu.Lock()
	defer clockMu.Unlock()
	now := time.Now().UTC().Truncate(time.Millisecond)
	if !now.After(clockLast) {
		now = clockLast.Add(time.Millisecond)
	}
	clockLast = now
	return now
}

const masteryColumns = `box, grad_step, best, integrative_ok, lapses, next_review_at, last_credit_at, last_fail_at, stale, queued, verify_queued, ease, updated_at`

// lockedMastery is an objective's mastery read for update. Mark is
// last_evidence_turn_id ("" = none): the turn whose evidence was saved last —
// the dispute conflict marker (only evidence and its restore move it).
type lockedMastery struct {
	M       learning.Mastery
	Updated time.Time
	Mark    string
	Exists  bool
}

// lockMastery reads an objective's mastery inside the caller's transaction; a
// missing row is the empty state (Exists=false).
func lockMastery(ctx context.Context, c conn, userID, objectiveID string) (lockedMastery, error) {
	var l lockedMastery
	err := c.row(ctx, `SELECT `+masteryColumns+`, COALESCE(last_evidence_turn_id, '') FROM user_objective_mastery
		WHERE user_id = $1 AND objective_id = $2`, userID, objectiveID).Scan(&l.M.Box, &l.M.GradStep, &l.M.Best,
		&l.M.IntegrativeOK, &l.M.Lapses, nts(&l.M.NextReviewAt), nts(&l.M.LastCreditAt), nts(&l.M.LastFailAt),
		&l.M.Stale, &l.M.Queued, &l.M.Verify, &l.M.Ease, ts(&l.Updated), &l.Mark)
	if errors.Is(err, sql.ErrNoRows) {
		return lockedMastery{M: learning.NewMastery()}, nil
	}
	l.Exists = err == nil
	return l, err
}

// saveMastery upserts an objective's mastery with the objective's current text
// and depth (the evidence's basis for change detection, design §2.5).
// mark is the evidence turn written as last_evidence_turn_id ("" keeps the
// current one). Returns the new updated_at.
func saveMastery(ctx context.Context, c conn, userID, objectiveID, grader string, m learning.Mastery, mark string) (time.Time, error) {
	var updated time.Time
	err := c.row(ctx, `
		INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, box, grad_step, best, integrative_ok, lapses,
			next_review_at, last_credit_at, last_fail_at, objective_text, objective_depth, stale, last_grader, queued, verify_queued,
			last_evidence_turn_id, ease, updated_at)
		SELECT $1, o.id, o.concept_id, $3, $4, $5, $6, $7, $8, $9, $10, o.text, o.depth, $11, $12, $14, $15, NULLIF($16, ''), $17, $13
		FROM objectives o WHERE o.id = $2
		ON CONFLICT (user_id, objective_id) DO UPDATE SET
			box = excluded.box, grad_step = excluded.grad_step, best = excluded.best,
			integrative_ok = excluded.integrative_ok, lapses = excluded.lapses, next_review_at = excluded.next_review_at,
			last_credit_at = excluded.last_credit_at, last_fail_at = excluded.last_fail_at,
			objective_text = excluded.objective_text, objective_depth = excluded.objective_depth,
			stale = excluded.stale, last_grader = excluded.last_grader, queued = excluded.queued,
			verify_queued = excluded.verify_queued, ease = excluded.ease, updated_at = excluded.updated_at,
			last_evidence_turn_id = COALESCE(excluded.last_evidence_turn_id, user_objective_mastery.last_evidence_turn_id)
		RETURNING updated_at`,
		userID, objectiveID, m.Box, m.GradStep, m.Best, m.IntegrativeOK, m.Lapses, m.NextReviewAt,
		m.LastCreditAt, m.LastFailAt, m.Stale, grader, clockNow(), m.Queued, m.Verify, mark, storedEase(m.Ease)).Scan(ts(&updated))
	if err != nil {
		return updated, fmt.Errorf("save mastery %s: %w", objectiveID, err)
	}
	return updated, nil
}

// storedEase is the ease column value: an unset (pre-v3 / neutral) ease is the default.
func storedEase(e float64) float64 {
	if e < learning.EaseMin || e > learning.EaseMax {
		return learning.EaseDefault
	}
	return e
}

// creditIntegrative sets integrative_ok on the covered objectives that already
// have evidence (the box does not move — design §4.2 N9).
func creditIntegrative(ctx context.Context, c conn, userID string, objectiveIDs []string) error {
	if len(objectiveIDs) == 0 {
		return nil
	}
	_, err := c.exec(ctx, `UPDATE user_objective_mastery SET integrative_ok = 1, updated_at = $3
		WHERE user_id = $1 AND objective_id IN (SELECT value FROM json_each($2)) AND NOT integrative_ok`, userID, objectiveIDs, clockNow())
	if err != nil {
		return fmt.Errorf("credit integrative: %w", err)
	}
	return nil
}

// ConceptMastery returns the mastery of the concept's active objectives
// (objectives without evidence are level 0). ErrNotFound when the concept is not visible.
func (s *SQLiteStore) ConceptMastery(ctx context.Context, userID, conceptID string) ([]models.ObjectiveMastery, error) {
	var visible bool
	if err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM concepts c WHERE c.id = $1 AND c.status <> 'retired' AND `+
		visibleTo("c", "$2")+`)`, conceptID, userID).Scan(&visible); err != nil {
		return nil, fmt.Errorf("concept mastery: %w", err)
	}
	if !visible {
		return nil, ErrNotFound
	}
	return s.objectiveMastery(ctx, userID, `o.concept_id = $2`, conceptID)
}

// ConceptsMastery is ConceptMastery for many concepts in one query (the map's
// rings): rows of invisible or retired concepts are left out, not an error.
func (s *SQLiteStore) ConceptsMastery(ctx context.Context, userID string, conceptIDs []string) ([]models.ObjectiveMastery, error) {
	if len(conceptIDs) == 0 {
		return []models.ObjectiveMastery{}, nil
	}
	return s.objectiveMastery(ctx, userID, `o.concept_id IN (SELECT c.id FROM concepts c
		WHERE c.id IN (SELECT value FROM json_each($2)) AND c.status <> 'retired' AND `+visibleTo("c", "$1")+`)`, conceptIDs)
}

// ObjectivesMastery returns the mastery of the given active objectives, in seed order.
func (s *SQLiteStore) ObjectivesMastery(ctx context.Context, userID string, objectiveIDs []string) ([]models.ObjectiveMastery, error) {
	if len(objectiveIDs) == 0 {
		return []models.ObjectiveMastery{}, nil
	}
	return s.objectiveMastery(ctx, userID, `o.id IN (SELECT value FROM json_each($2))`, objectiveIDs)
}

func (s *SQLiteStore) objectiveMastery(ctx context.Context, userID, where string, arg any) ([]models.ObjectiveMastery, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT o.id, o.concept_id, o.text, o.depth, o.ordinal,
			COALESCE(m.box, 0), COALESCE(m.grad_step, 0), COALESCE(m.best, 'none'), COALESCE(m.integrative_ok, 0),
			COALESCE(m.lapses, 0), m.next_review_at, COALESCE(m.stale, 0), COALESCE(m.queued, 0),
			COALESCE(m.last_grader, ''), m.updated_at
		FROM objectives o
		LEFT JOIN user_objective_mastery m ON m.objective_id = o.id AND m.user_id = $1
		WHERE o.status = 'active' AND `+where+`
		ORDER BY o.concept_id, o.ordinal`, userID, arg), func(row rowScanner) (models.ObjectiveMastery, error) {
		var o models.ObjectiveMastery
		err := row.Scan(&o.ObjectiveID, &o.ConceptID, &o.Text, &o.Depth, &o.Ordinal, &o.Box, &o.GradStep, &o.Best,
			&o.IntegrativeOK, &o.Lapses, nts(&o.NextReviewAt), &o.Stale, &o.Queued, &o.LastGrader, nts(&o.UpdatedAt))
		o.Level = learning.DisplayLevel(learning.Mastery{Box: o.Box, Best: o.Best, IntegrativeOK: o.IntegrativeOK})
		return o, err
	})
	if err != nil {
		return nil, fmt.Errorf("objective mastery: %w", err)
	}
	return list, nil
}

// ReopenObjective is '다시 배우기' (design §2.4), in one transaction: the
// objective's read state is removed (it can be generated/learned again), the
// concept goes back to in_progress directly (bypassing SetConceptState, so a
// completed/known concept changes too), and the mastery drops to box 0 with
// lapses 0 and no review date, not waiting in the inflow or verification queue (best is kept). The evidence
// marker moves to a fresh value, so a later accepted dispute of an earlier grade is a conflict. Returns the objective's concept.
func (s *SQLiteStore) ReopenObjective(ctx context.Context, userID, objectiveID string) (string, error) {
	var conceptID string
	err := s.tx(ctx, func(c conn) error {
		err := c.row(ctx, `SELECT o.concept_id FROM objectives o JOIN concepts c ON c.id = o.concept_id
			WHERE o.id = $1 AND o.status = 'active' AND c.status <> 'retired' AND `+visibleTo("c", "$2"), objectiveID, userID).Scan(&conceptID)
		if errors.Is(err, sql.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("load objective: %w", err)
		}
		if _, err := c.exec(ctx, `DELETE FROM user_objective_states WHERE user_id = $1 AND objective_id = $2`, userID, objectiveID); err != nil {
			return fmt.Errorf("reopen objective state: %w", err)
		}
		if _, err := c.exec(ctx, `
			INSERT INTO user_concept_states (user_id, concept_id, state, updated_at) VALUES ($1, $2, 'in_progress', NOW())
			ON CONFLICT (user_id, concept_id) DO UPDATE SET state = 'in_progress', updated_at = NOW()`, userID, conceptID); err != nil {
			return fmt.Errorf("reopen concept state: %w", err)
		}
		// 근거 표지를 새 값으로 바꾼다: 다시 배우기 전의 판정에 대한 이의가 받아들여져도 초기화를 덮지 않고 충돌로 끝난다.
		if _, err := c.exec(ctx, `UPDATE user_objective_mastery SET box = 0, grad_step = 0, lapses = 0, next_review_at = NULL,
			queued = 0, verify_queued = 0, last_evidence_turn_id = $4, updated_at = $3 WHERE user_id = $1 AND objective_id = $2`,
			userID, objectiveID, clockNow(), NewID()); err != nil {
			return fmt.Errorf("reopen mastery: %w", err)
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return conceptID, nil
}

// markObjectiveChanges runs right after the seed upsert, in the same
// transaction (design §2.5): items and evidence written for an objective text
// or depth that changed since are retired / marked stale (box ≤ 1, no
// integrative credit — learning.MarkStale). The same EXISTS statements as
// witems (LEAST → MIN, clock_timestamp() → clockNow).
func markObjectiveChanges(ctx context.Context, c conn) error {
	if _, err := c.exec(ctx, `
		UPDATE check_items SET status = 'retired'
		WHERE status = 'active' AND objective_id IS NOT NULL AND EXISTS (
			SELECT 1 FROM objectives o WHERE o.id = check_items.objective_id
			  AND (o.text <> check_items.objective_text OR o.depth <> check_items.objective_depth))`); err != nil {
		return fmt.Errorf("retire changed check items: %w", err)
	}
	if _, err := c.exec(ctx, `
		UPDATE user_objective_mastery SET stale = 1, box = MIN(box, 1), grad_step = 0, integrative_ok = 0,
			updated_at = $1
		WHERE NOT stale AND EXISTS (
			SELECT 1 FROM objectives o WHERE o.id = user_objective_mastery.objective_id
			  AND (o.text <> user_objective_mastery.objective_text OR o.depth <> user_objective_mastery.objective_depth))`, clockNow()); err != nil {
		return fmt.Errorf("mark stale mastery: %w", err)
	}
	return nil
}
