package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/check"
	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// DiagnosticItems is a validated pretest / known verification / placement set
// to store (design §5 P3). The items belong to no lecture (lecture_id NULL, so
// lecture sets and their keys never see them); each item's concept is its
// objective's concept. CreatedFor is the user; PromptHash is the user's
// check.pretest hash ("" = default prompt).
type DiagnosticItems struct {
	PromptHash    string
	CreatedFor    string
	JobID         string
	Model         string
	PromptVersion string
	Items         []check.Item
}

// SaveDiagnosticItems inserts the set in one transaction, in order.
// witems' INSERT … RETURNING CTE → insert with store.NewID(), then read back.
func (s *SQLiteStore) SaveDiagnosticItems(ctx context.Context, d DiagnosticItems) ([]CheckItem, error) {
	out := make([]CheckItem, 0, len(d.Items))
	err := s.tx(ctx, func(c conn) error {
		for _, it := range d.Items {
			rubric, err := json.Marshal(it.Rubric)
			if err != nil {
				return err
			}
			id := NewID()
			res, err := c.exec(ctx, `
				INSERT INTO check_items (id, concept_id, objective_id, purpose, format, prompt, material, rubric,
					objective_text, objective_depth, prompt_hash, created_for, generation_job_id, model, prompt_version)
				SELECT $12, o.concept_id, o.id, $2, $3, $4, $5, $6, o.text, o.depth, $7, NULLIF($8, ''), NULLIF($9, ''), $10, $11
				FROM objectives o WHERE o.id = $1`,
				it.ObjectiveID, it.Purpose, it.Format, it.Prompt, nonNullJSON(it.Material, "{}"), string(rubric),
				d.PromptHash, d.CreatedFor, d.JobID, d.Model, d.PromptVersion, id)
			if err != nil {
				return fmt.Errorf("insert diagnostic item: %w", err)
			}
			if n, _ := res.RowsAffected(); n == 0 {
				return fmt.Errorf("diagnostic item objective %s: %w", it.ObjectiveID, ErrNotFound)
			}
			saved, err := scanCheckItem(c.row(ctx, `SELECT `+checkItemColumns+` FROM check_items ci WHERE ci.id = $1`, id))
			if err != nil {
				return fmt.Errorf("load diagnostic item: %w", err)
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

// QueueKnownVerify puts the concept's active objectives without evidence into
// the verification queue (bulk '이미 앎'): a mastery row with best none, no
// review date and verify_queued. Objectives with evidence are left alone.
// clock_timestamp() → clockNow() (one stamp for the batch; rows are different objectives).
func (s *SQLiteStore) QueueKnownVerify(ctx context.Context, userID, conceptID string) (int64, error) {
	res, err := s.c().exec(ctx, `
		INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, objective_text, objective_depth, verify_queued, updated_at)
		SELECT $1, o.id, o.concept_id, o.text, o.depth, 1, $3
		FROM objectives o JOIN concepts c ON c.id = o.concept_id
		WHERE o.concept_id = $2 AND o.status = 'active' AND c.status <> 'retired' AND `+visibleTo("c", "$1")+`
		ON CONFLICT (user_id, objective_id) DO UPDATE SET verify_queued = 1, updated_at = excluded.updated_at
		WHERE user_objective_mastery.best = 'none' AND NOT user_objective_mastery.verify_queued`, userID, conceptID, clockNow())
	if err != nil {
		return 0, fmt.Errorf("queue known verify: %w", err)
	}
	return res.RowsAffected()
}

// verifyScope: a queued objective (mastery row m of user $1) on a visible,
// active concept the user still marks as known (undoing '이미 앎' drops it).
var verifyScope = reviewScope + `
	JOIN user_concept_states us ON us.user_id = $1 AND us.concept_id = m.concept_id AND us.state = 'known'`

// unseenVerifyItem: an active known_verify item of the objective the user was never asked.
var unseenVerifyItem = `ci.objective_id = m.objective_id AND ci.purpose = 'known_verify' AND ci.status = 'active' AND ` +
	checkItemVisible + ` AND NOT EXISTS (SELECT 1 FROM check_turns t WHERE t.user_id = $1 AND t.item_id = ci.id)`

// KnownVerifyPicks returns up to limit queued objectives that have an unseen
// known_verify item, oldest in the queue first, with that item.
// DISTINCT ON (objective_id) → ROW_NUMBER() OVER (PARTITION BY …) = 1 (item columns re-joined).
func (s *SQLiteStore) KnownVerifyPicks(ctx context.Context, userID string, limit int) ([]ReviewPick, error) {
	if limit <= 0 {
		return []ReviewPick{}, nil
	}
	list, err := collect(s.c().query(ctx, `
		SELECT `+checkItemColumns+` FROM (
			SELECT ci.id, m.objective_id, m.updated_at AS queued_at,
				ROW_NUMBER() OVER (PARTITION BY m.objective_id ORDER BY ci.created_at, ci.rowid) AS rn
			FROM user_objective_mastery m`+verifyScope+`
			JOIN check_items ci ON `+unseenVerifyItem+`
			WHERE m.user_id = $1 AND m.verify_queued) p
		JOIN check_items ci ON ci.id = p.id
		WHERE p.rn = 1
		ORDER BY p.queued_at, p.objective_id LIMIT $2`, userID, limit), func(row rowScanner) (ReviewPick, error) {
		it, err := scanCheckItem(row)
		return ReviewPick{Item: it}, err
	})
	if err != nil {
		return nil, fmt.Errorf("known verify picks: %w", err)
	}
	return list, nil
}

// KnownVerifyUsed counts the known_verify items already in the user's sessions of a key (today's reviews).
func (s *SQLiteStore) KnownVerifyUsed(ctx context.Context, userID, sessionKey string) (int, error) {
	var n int
	err := s.c().row(ctx, `
		SELECT COUNT(*) FROM check_sessions cs
		JOIN json_each(cs.item_ids) e
		JOIN check_items ci ON ci.id = e.value AND ci.purpose = 'known_verify'
		WHERE cs.user_id = $1 AND cs.session_key = $2`, userID, sessionKey).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("known verify used: %w", err)
	}
	return n, nil
}

// KnownVerifyReady is how many verifications today's review can still mix in:
// KnownVerifyPerDay minus those already in today's sessions, bounded by the
// queued objectives with an unseen item.
func (s *SQLiteStore) KnownVerifyReady(ctx context.Context, userID string, now time.Time) (int, error) {
	used, err := s.KnownVerifyUsed(ctx, userID, learning.ReviewSessionKey(now))
	if err != nil {
		return 0, err
	}
	left := learning.KnownVerifyPerDay - used
	if left <= 0 {
		return 0, nil
	}
	picks, err := s.KnownVerifyPicks(ctx, userID, left)
	return len(picks), err
}

// KnownVerifyTarget is a concept whose queued objectives need a known_verify item.
type KnownVerifyTarget struct {
	ConceptID    string
	ObjectiveIDs []string
}

// KnownVerifyTargets lists concepts (oldest queued first) with queued
// objectives lacking an unseen known_verify item — what the scheduler writes next.
// array_agg(… ORDER BY) → ordered rows grouped here (concept order = MIN(updated_at) window).
func (s *SQLiteStore) KnownVerifyTargets(ctx context.Context, userID string, limit int) ([]KnownVerifyTarget, error) {
	type row struct{ concept, objective string }
	rows, err := collect(s.c().query(ctx, `
		SELECT m.concept_id, m.objective_id
		FROM user_objective_mastery m`+verifyScope+`
		WHERE m.user_id = $1 AND m.verify_queued AND NOT EXISTS (SELECT 1 FROM check_items ci WHERE `+unseenVerifyItem+`)
		ORDER BY MIN(m.updated_at) OVER (PARTITION BY m.concept_id), m.concept_id, o.depth, o.ordinal`, userID),
		func(r rowScanner) (row, error) {
			var x row
			err := r.Scan(&x.concept, &x.objective)
			return x, err
		})
	if err != nil {
		return nil, fmt.Errorf("known verify targets: %w", err)
	}
	out := []KnownVerifyTarget{}
	for _, r := range rows {
		if n := len(out); n > 0 && out[n-1].ConceptID == r.concept {
			out[n-1].ObjectiveIDs = append(out[n-1].ObjectiveIDs, r.objective)
			continue
		}
		if len(out) == limit {
			break
		}
		out = append(out, KnownVerifyTarget{ConceptID: r.concept, ObjectiveIDs: []string{r.objective}})
	}
	return out, nil
}

// PretestAnswers returns the user's first pretest answer on each of the
// objectives ('처음 예측', design §5 P3); objectives never predicted are absent.
// DISTINCT ON (objective_id) → ROW_NUMBER() OVER (PARTITION BY …) = 1.
func (s *SQLiteStore) PretestAnswers(ctx context.Context, userID string, objectiveIDs []string) ([]models.Prediction, error) {
	if len(objectiveIDs) == 0 {
		return []models.Prediction{}, nil
	}
	list, err := collect(s.c().query(ctx, `
		SELECT objective_id, prompt, text, dont_know, created_at FROM (
			SELECT ci.objective_id, ci.prompt, t.text, t.move = 'dont_know' AS dont_know, t.created_at,
				ROW_NUMBER() OVER (PARTITION BY ci.objective_id ORDER BY t.created_at, t.seq) AS rn
			FROM check_turns t
			JOIN check_sessions cs ON cs.id = t.session_id AND cs.kind = 'pretest'
			JOIN check_items ci ON ci.id = t.item_id AND ci.purpose = 'pretest'
			WHERE t.user_id = $1 AND t.role = 'learner' AND t.move IN ('answer', 'dont_know')
			  AND ci.objective_id IN (SELECT value FROM json_each($2)))
		WHERE rn = 1
		ORDER BY objective_id`, userID, objectiveIDs), func(row rowScanner) (models.Prediction, error) {
		var p models.Prediction
		err := row.Scan(&p.ObjectiveID, &p.Prompt, &p.Answer, &p.DontKnow, ts(&p.At))
		return p, err
	})
	if err != nil {
		return nil, fmt.Errorf("pretest answers: %w", err)
	}
	return list, nil
}

// PretestSeen: one of the objectives was already in a pretest session of the
// user (any status but failed) — the prediction is asked only once.
func (s *SQLiteStore) PretestSeen(ctx context.Context, userID string, objectiveIDs []string) (bool, error) {
	var seen bool
	err := s.c().row(ctx, `
		SELECT EXISTS (SELECT 1 FROM check_sessions cs
			JOIN json_each(cs.item_ids) e
			JOIN check_items ci ON ci.id = e.value
			WHERE cs.user_id = $1 AND cs.kind = 'pretest' AND cs.status <> 'failed'
			  AND ci.objective_id IN (SELECT value FROM json_each($2)))`,
		userID, objectiveIDs).Scan(&seen)
	if err != nil {
		return false, fmt.Errorf("pretest seen: %w", err)
	}
	return seen, nil
}

// PlacementConcepts returns the concepts of the user's chapter in chapter
// order (the applied draft's, prerequisites first) with depth, the user's
// state and the first active objective. ErrNotFound when the chapter is not the user's.
// jsonb_array_elements WITH ORDINALITY → json_each (key + 1 = 1-based order),
// LEFT JOIN LATERAL (… LIMIT 1) → a correlated subquery.
func (s *SQLiteStore) PlacementConcepts(ctx context.Context, userID, chapterID string) ([]learning.PlacementConcept, error) {
	var one int
	err := s.c().row(ctx, `SELECT 1 FROM chapters WHERE id = $1 AND user_id = $2`, chapterID, userID).Scan(&one)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("load chapter: %w", err)
	}
	list, err := collect(s.c().query(ctx, `
		SELECT c.id, COALESCE((SELECT o.id FROM objectives o WHERE o.concept_id = c.id AND o.status = 'active'
				ORDER BY o.depth, o.ordinal LIMIT 1), ''),
			c.depth, d.key + 1, COALESCE(us.state, '')
		FROM chapter_drafts cd
		JOIN json_each(cd.draft, '$.concepts') d
		JOIN concepts c ON c.id = json_extract(d.value, '$.id') AND c.status = 'active' AND `+visibleTo("c", "$1")+`
		LEFT JOIN user_concept_states us ON us.user_id = $1 AND us.concept_id = c.id
		WHERE cd.user_id = $1 AND cd.chapter_id = $2
		ORDER BY d.key`, userID, chapterID), func(row rowScanner) (learning.PlacementConcept, error) {
		var c learning.PlacementConcept
		err := row.Scan(&c.ConceptID, &c.ObjectiveID, &c.Depth, &c.Order, &c.State)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("placement concepts: %w", err)
	}
	return list, nil
}
