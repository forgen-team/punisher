package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// reviewScope joins a mastery row `m` (user $1) to its active objective and a
// non-retired concept visible to the user (design §2.5, §5 P2).
var reviewScope = `
	JOIN objectives o ON o.id = m.objective_id AND o.status = 'active'
	JOIN concepts c ON c.id = m.concept_id AND c.status <> 'retired' AND ` + visibleTo("c", "$1")

// ReviewQueue returns the user's due objectives (design §5 P2): reviewed by
// now, not waiting (queued), not graduated, on an active concept/objective.
// Unordered — learning.OrderReviews ranks them.
func (s *SQLiteStore) ReviewQueue(ctx context.Context, userID string, now time.Time) ([]learning.ReviewCandidate, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT m.objective_id, m.concept_id, c.area, m.box, m.grad_step, m.next_review_at
		FROM user_objective_mastery m`+reviewScope+`
		WHERE m.user_id = $1 AND m.next_review_at <= $2 AND NOT m.queued AND m.grad_step < 2`, userID, now),
		func(row rowScanner) (learning.ReviewCandidate, error) {
			var c learning.ReviewCandidate
			err := row.Scan(&c.ObjectiveID, &c.ConceptID, &c.Area, &c.Box, &c.GradStep, ts(&c.NextReviewAt))
			return c, err
		})
	if err != nil {
		return nil, fmt.Errorf("review queue: %w", err)
	}
	return list, nil
}

// COUNT(*) FILTER (WHERE …) → COUNT(CASE WHEN … THEN 1 END) (0, not NULL, on no rows).
// Timestamps are UTC TEXT in one format, so the comparisons are lexicographic;
// $3 is the KST day start rendered in UTC (learning.KSTDayStart).
var reviewCountsSQL = `
	SELECT COUNT(CASE WHEN NOT m.queued AND m.next_review_at <= $2 THEN 1 END),
	       COUNT(CASE WHEN NOT m.queued AND m.next_review_at < $3 THEN 1 END),
	       COUNT(CASE WHEN m.queued THEN 1 END)
	FROM user_objective_mastery m` + reviewScope + `
	WHERE m.user_id = $1 AND m.grad_step < 2`

// ReviewCounts counts due (≤ now), overdue (due before today, KST) and queued
// objectives, and the known verifications today's review can mix in.
func (s *SQLiteStore) ReviewCounts(ctx context.Context, userID string, now time.Time) (learning.ReviewCounts, error) {
	n, err := reviewCounts(ctx, s.c(), userID, now)
	if err != nil {
		return n, err
	}
	n.Verify, err = s.KnownVerifyReady(ctx, userID, now)
	return n, err
}

func reviewCounts(ctx context.Context, c conn, userID string, now time.Time) (learning.ReviewCounts, error) {
	var n learning.ReviewCounts
	err := c.row(ctx, reviewCountsSQL, userID, now, learning.KSTDayStart(now)).Scan(&n.Due, &n.Overdue, &n.Queued)
	if err != nil {
		return n, fmt.Errorf("review counts: %w", err)
	}
	return n, nil
}

// ReleaseQueuedReviews lets waiting objectives into the queue (design §2.6):
// the due count is taken, learning.ReleaseCount decides how many, and the
// oldest waiting ones (by last evidence) get their review date
// (learning.NextDue). Releasing is not evidence: updated_at is kept.
// witems takes a per-user advisory lock and FOR UPDATE; here the transaction
// alone serialises it on the single connection.
func (s *SQLiteStore) ReleaseQueuedReviews(ctx context.Context, userID string, throughput int, now time.Time, intervals []int) (int, error) {
	released := 0
	err := s.tx(ctx, func(c conn) error {
		counts, err := reviewCounts(ctx, c, userID, now)
		if err != nil {
			return err
		}
		n := learning.ReleaseCount(counts.Due, throughput)
		if n == 0 || counts.Queued == 0 {
			return nil
		}
		type waiting struct {
			id string
			m  learning.Mastery
		}
		list, err := collect(c.query(ctx, `SELECT m.objective_id, m.box, m.grad_step FROM user_objective_mastery m`+reviewScope+`
			WHERE m.user_id = $1 AND m.queued AND m.grad_step < 2
			ORDER BY m.updated_at, m.objective_id LIMIT $2`, userID, n), func(row rowScanner) (waiting, error) {
			var w waiting
			err := row.Scan(&w.id, &w.m.Box, &w.m.GradStep)
			return w, err
		})
		if err != nil {
			return fmt.Errorf("queued reviews: %w", err)
		}
		for _, w := range list {
			if _, err := c.exec(ctx, `UPDATE user_objective_mastery SET queued = 0, next_review_at = $3
				WHERE user_id = $1 AND objective_id = $2`, userID, w.id, learning.NextDue(w.m, now, intervals)); err != nil {
				return fmt.Errorf("release review: %w", err)
			}
		}
		released = len(list)
		return nil
	})
	if err != nil {
		return 0, err
	}
	return released, nil
}

// ReviewPick is the item chosen for a due objective.
type ReviewPick struct {
	Item CheckItem
	// Reused: every visible item of the objective was seen; this is the one seen longest ago.
	Reused bool
}

// ReviewItems picks one item per objective (design §5 P2): an item of the
// objective (check, review or known_verify purpose) the user has never been asked, else the
// one asked longest ago (Reused). Objectives without a visible item are absent.
// DISTINCT ON (objective_id) … ORDER BY → ROW_NUMBER() OVER (PARTITION BY …) = 1.
func (s *SQLiteStore) ReviewItems(ctx context.Context, userID string, objectiveIDs []string) (map[string]ReviewPick, error) {
	out := map[string]ReviewPick{}
	if len(objectiveIDs) == 0 {
		return out, nil
	}
	picks, err := collect(s.c().query(ctx, `
		SELECT `+checkItemColumns+`, r.seen
		FROM (
			SELECT ci.id, seen.at IS NOT NULL AS seen,
				ROW_NUMBER() OVER (PARTITION BY ci.objective_id
					ORDER BY seen.at ASC NULLS FIRST, (ci.purpose = 'review') DESC, ci.created_at, ci.id) AS rn
			FROM check_items ci
			LEFT JOIN (SELECT t.item_id, MAX(t.created_at) AS at FROM check_turns t WHERE t.user_id = $1 GROUP BY t.item_id) seen
				ON seen.item_id = ci.id
			WHERE ci.objective_id IN (SELECT value FROM json_each($2)) AND ci.status = 'active'
			  AND ci.purpose IN ('check', 'review', 'known_verify') AND `+checkItemVisible+`) r
		JOIN check_items ci ON ci.id = r.id
		WHERE r.rn = 1`, userID, objectiveIDs), func(row rowScanner) (ReviewPick, error) {
		var p ReviewPick
		it, err := scanCheckItem(reviewPickScanner{row, &p.Reused})
		p.Item = it
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("review items: %w", err)
	}
	for _, p := range picks {
		out[p.Item.ObjectiveID] = p
	}
	return out, nil
}

// reviewPickScanner lets scanCheckItem read the item columns and appends the
// trailing "seen" flag of a ReviewItems row.
type reviewPickScanner struct {
	row    rowScanner
	reused *bool
}

func (r reviewPickScanner) Scan(dest ...any) error {
	return r.row.Scan(append(dest, r.reused)...)
}

// ItemSeenBefore: the user was asked the item in another session (a reused review item).
func (s *SQLiteStore) ItemSeenBefore(ctx context.Context, userID, itemID, sessionID string) (bool, error) {
	var seen bool
	err := s.c().row(ctx, `SELECT EXISTS (SELECT 1 FROM check_turns
		WHERE user_id = $1 AND item_id = $2 AND session_id <> $3)`, userID, itemID, sessionID).Scan(&seen)
	if err != nil {
		return false, fmt.Errorf("item seen: %w", err)
	}
	return seen, nil
}

// LatestCheckSession is the user's newest session for a key (any status); ErrNotFound when none.
func (s *SQLiteStore) LatestCheckSession(ctx context.Context, userID, sessionKey string) (models.CheckSession, error) {
	c, err := scanSession(s.c().row(ctx, `SELECT `+sessionColumns+` FROM check_sessions
		WHERE user_id = $1 AND session_key = $2 ORDER BY created_at DESC, rowid DESC LIMIT 1`, userID, sessionKey))
	if errors.Is(err, sql.ErrNoRows) {
		return c, ErrNotFound
	}
	return c, err
}

// AbandonStaleReviews abandons open review sessions of an earlier day
// (session_key other than todayKey); evidence already applied stays.
// userID "" = every user (scheduler sweep).
func (s *SQLiteStore) AbandonStaleReviews(ctx context.Context, userID, todayKey string) (int64, error) {
	res, err := s.c().exec(ctx, `
		UPDATE check_sessions SET status = 'abandoned', finished_at = NOW(), updated_at = NOW()
		WHERE kind = 'review' AND status IN ('preparing', 'active') AND session_key <> $2 AND ($1 = '' OR user_id = $1)`,
		userID, todayKey)
	if err != nil {
		return 0, fmt.Errorf("abandon stale reviews: %w", err)
	}
	return affected(res), nil
}
