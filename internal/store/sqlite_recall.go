package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/forgen-team/punisher/internal/check"
	"github.com/forgen-team/punisher/internal/learning"
)

// RecallPromptHash marks the review items made from in-lecture recall prompts
// (never part of a lecture's shared check set; always the learner's own).
const RecallPromptHash = "recall"

// RecallReview is a recall prompt the learner answered, to be reviewed later.
type RecallReview struct {
	UserID, LectureID, ConceptID, ObjectiveID string
	Anchor, Question, Key                     string
	// Due: when the objective's first review is (a day after reading).
	Due time.Time
	// Hold: the review backlog is over its limit — the objective waits (queued) like a new check's.
	Hold bool
	// PromptVersion recorded on the item.
	PromptVersion string
}

// AddRecallReview puts a recall prompt into the learner's review bank as a
// self-gradable review item of its objective (one per lecture block and
// question), and gives the objective its first review date — Due, or queued
// when Hold — only when it has none yet and no evidence (an objective already
// in the review schedule keeps its date; the item just joins its bank). It is
// not evidence: box, best and credit dates are untouched. Returns the item id
// and whether the objective was scheduled.
func (s *SQLiteStore) AddRecallReview(ctx context.Context, r RecallReview) (string, bool, error) {
	var itemID string
	scheduled := false
	err := s.tx(ctx, func(c conn) error {
		err := c.row(ctx, `SELECT id FROM check_items
			WHERE lecture_id = $1 AND prompt_hash = $2 AND created_for = $3 AND status = 'active'
			  AND json_extract(rubric, '$.anchor') = $4 AND prompt = $5 LIMIT 1`,
			r.LectureID, RecallPromptHash, r.UserID, r.Anchor, r.Question).Scan(&itemID)
		if errors.Is(err, sql.ErrNoRows) {
			item := check.Item{ObjectiveID: r.ObjectiveID, Purpose: check.PurposeReview, Format: check.FormatRecall, Prompt: r.Question,
				Rubric: check.Rubric{Must: []string{r.Key}, Nice: []string{}, Misconceptions: []string{}, Answer: r.Key, Anchor: r.Anchor, Covers: []string{}}}
			saved, ierr := insertCheckItem(ctx, c, CheckItemKey{LectureID: r.LectureID, Purpose: check.PurposeReview,
				PromptHash: RecallPromptHash, CreatedFor: r.UserID}, r.ConceptID, "", "", r.PromptVersion, item, "")
			itemID, err = saved.ID, ierr
		}
		if err != nil {
			return fmt.Errorf("recall review item: %w", err)
		}
		var due any = r.Due.UTC()
		if r.Hold {
			due = nil
		}
		res, err := c.exec(ctx, `
			INSERT INTO user_objective_mastery (user_id, objective_id, concept_id, objective_text, objective_depth, next_review_at, queued, updated_at)
			SELECT $1, o.id, o.concept_id, o.text, o.depth, $3, $4, $5 FROM objectives o WHERE o.id = $2 AND o.status = 'active'
			ON CONFLICT (user_id, objective_id) DO UPDATE SET next_review_at = excluded.next_review_at, queued = excluded.queued
			WHERE user_objective_mastery.next_review_at IS NULL AND NOT user_objective_mastery.queued
			  AND user_objective_mastery.grad_step < 2 AND user_objective_mastery.best = 'none'`,
			r.UserID, r.ObjectiveID, due, r.Hold, clockNow())
		if err != nil {
			return fmt.Errorf("schedule recall review: %w", err)
		}
		scheduled = affected(res) > 0
		return nil
	})
	return itemID, scheduled, err
}

// SetAttemptReviewItem records the review item an attempt put into the queue.
func (s *SQLiteStore) SetAttemptReviewItem(ctx context.Context, userID, attemptID, itemID string) error {
	if _, err := s.c().exec(ctx, `UPDATE lecture_exercise_attempts SET review_item_id = $3 WHERE user_id = $1 AND id = $2`,
		userID, attemptID, itemID); err != nil {
		return fmt.Errorf("set attempt review item: %w", err)
	}
	return nil
}

// LectureAnchorSignals are the block-level signals of the user's latest
// attempts (confident wrong predictions, forgotten recalls) and the lecture
// questions per block anchor (research #15).
func (s *SQLiteStore) LectureAnchorSignals(ctx context.Context, userID, lectureID string) ([]learning.AnchorSignal, error) {
	byAnchor := map[string]*learning.AnchorSignal{}
	var order []string
	get := func(anchor string) *learning.AnchorSignal {
		if byAnchor[anchor] == nil {
			byAnchor[anchor] = &learning.AnchorSignal{Anchor: anchor}
			order = append(order, anchor)
		}
		return byAnchor[anchor]
	}
	attempts, err := s.LatestExerciseAttempts(ctx, userID, lectureID)
	if err != nil {
		return nil, err
	}
	for _, a := range attempts {
		switch {
		case a.Kind == learning.AttemptPause && a.Verdict == learning.ExerciseIncorrect && a.Confidence == learning.ConfidenceSure:
			get(a.Anchor).ConfidentMiss = true
		case a.Kind == learning.AttemptRecall && a.Verdict == learning.RecallForgot:
			get(a.Anchor).Forgot = true
		}
	}
	type count struct {
		anchor string
		n      int
	}
	counts, err := collect(s.c().query(ctx, `SELECT anchor, COUNT(*) FROM lecture_questions
		WHERE user_id = $1 AND lecture_id = $2 AND anchor <> '' GROUP BY anchor ORDER BY anchor`, userID, lectureID), func(r rowScanner) (count, error) {
		var c count
		err := r.Scan(&c.anchor, &c.n)
		return c, err
	})
	if err != nil {
		return nil, fmt.Errorf("lecture anchor signals: %w", err)
	}
	for _, c := range counts {
		get(c.anchor).Questions += c.n
	}
	out := make([]learning.AnchorSignal, 0, len(order))
	for _, a := range order {
		out = append(out, *byAnchor[a])
	}
	return out, nil
}

// QualityRecord is the stored quality of a lecture's latest review.
type QualityRecord struct {
	LectureID, ConceptID, PromptVersion, Kind, Status string
	Quality                                           json.RawMessage
}

// QualityRecords returns the latest review quality of every lecture that has
// one (reviews recorded before quality scores are left out), oldest first.
func (s *SQLiteStore) QualityRecords(ctx context.Context) ([]QualityRecord, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT l.id, l.concept_id, l.prompt_version, l.kind, l.status, r.quality FROM lectures l
		JOIN lecture_reviews r ON r.id = (SELECT r2.id FROM lecture_reviews r2 WHERE r2.lecture_id = l.id AND r2.quality <> '{}'
			ORDER BY r2.created_at DESC LIMIT 1)
		ORDER BY l.created_at, l.id`), func(row rowScanner) (QualityRecord, error) {
		var q QualityRecord
		err := row.Scan(&q.LectureID, &q.ConceptID, &q.PromptVersion, &q.Kind, &q.Status, rawJSON(&q.Quality))
		return q, err
	})
	if err != nil {
		return nil, fmt.Errorf("quality records: %w", err)
	}
	return list, nil
}

// RepeatedMisconceptions are the misconceptions the check tutor found at
// least minCount times in the user's answers on the concept or its
// prerequisites, most frequent first (acknowledged ones go through OpenMisconceptions).
func (s *SQLiteStore) RepeatedMisconceptions(ctx context.Context, userID, conceptID string, minCount, limit int) ([]string, error) {
	list, err := collect(s.c().query(ctx, `
		SELECT json_extract(t.grade, '$.misconception') AS m FROM check_turns t JOIN check_items ci ON ci.id = t.item_id
		WHERE t.user_id = $1 AND (ci.concept_id = $2 OR ci.concept_id IN (
			SELECT e.from_id FROM concept_edges e WHERE e.to_id = $2 AND e.kind = 'prerequisite'))
		  AND TRIM(COALESCE(json_extract(t.grade, '$.misconception'), '')) <> ''
		  AND NOT EXISTS (SELECT 1 FROM user_misconceptions um WHERE um.user_id = $1 AND um.turn_id = t.id)
		GROUP BY m HAVING COUNT(*) >= $3 ORDER BY COUNT(*) DESC, m LIMIT $4`, userID, conceptID, minCount, limit),
		func(row rowScanner) (string, error) {
			var m string
			err := row.Scan(&m)
			return m, err
		})
	if err != nil {
		return nil, fmt.Errorf("repeated misconceptions: %w", err)
	}
	return list, nil
}

// UnanchoredPretestItems are the user's pretest items of the objectives,
// created since (an instant lecture's job start) and pointing to no lecture block yet.
func (s *SQLiteStore) UnanchoredPretestItems(ctx context.Context, userID string, objectiveIDs []string, since time.Time) ([]CheckItem, error) {
	list, err := collect(s.c().query(ctx, `SELECT `+checkItemColumns+` FROM check_items ci
		WHERE ci.purpose = 'pretest' AND ci.created_for = $1 AND ci.objective_id IN (SELECT value FROM json_each($2))
		  AND ci.created_at >= $3 AND COALESCE(json_extract(ci.rubric, '$.anchor'), '') = ''
		ORDER BY ci.created_at, ci.rowid`, userID, nonNil(objectiveIDs), since.UTC()), scanCheckItem)
	if err != nil {
		return nil, fmt.Errorf("unanchored pretest items: %w", err)
	}
	return list, nil
}

// SetCheckItemAnchor points one of the user's own items to a lecture block.
func (s *SQLiteStore) SetCheckItemAnchor(ctx context.Context, userID, itemID, anchor string) error {
	if _, err := s.c().exec(ctx, `UPDATE check_items SET rubric = json_set(rubric, '$.anchor', $3) WHERE id = $1 AND created_for = $2`,
		itemID, userID, anchor); err != nil {
		return fmt.Errorf("set check item anchor: %w", err)
	}
	return nil
}
