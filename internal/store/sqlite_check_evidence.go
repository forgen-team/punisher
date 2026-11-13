package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
)

// ItemFinish closes the current item of a session: its outcome is decided and
// applied to the objective's mastery, the last judgement turn is written with
// the before-snapshot, and the cursor moves on — all in one transaction, so
// each item's evidence is saved the moment it ends (design §4.3).
type ItemFinish struct {
	UserID    string
	SessionID string
	Item      CheckItem
	Result    learning.ItemResult
	// IntegrativeOK: the integrative item's challenge was defended (credit for Item.Rubric.Covers).
	IntegrativeOK bool
	// Turn is the item's last judgement (ID set = complete/rewrite that turn, else insert).
	Turn TurnWrite
	// NextAsk asks the next item (nil = this was the last item).
	NextAsk *TurnWrite
	// Abandon closes the session after this item ('포기' with a half-answered item).
	Abandon bool
	// Hold: the review backlog is over the inflow limit — an objective entering
	// the review schedule from this item waits (queued, design §2.6).
	Hold      bool
	Intervals []int
	Now       time.Time
}

// ResultEntry is one item's line of the session result (check_sessions.result.items).
type ResultEntry struct {
	ItemID        string     `json:"itemId"`
	ObjectiveID   string     `json:"objectiveId"`
	Outcome       string     `json:"outcome"`
	LevelBefore   int        `json:"levelBefore"`
	LevelAfter    int        `json:"levelAfter"`
	Missing       []string   `json:"missing"`
	Anchor        string     `json:"anchor"`
	Misconception string     `json:"misconception"`
	NextReviewAt  *time.Time `json:"nextReviewAt"`
	Lapses        int        `json:"lapses"`
	// Queued: the objective waits for the review queue ("복습이 밀려 대기").
	Queued bool `json:"queued"`
}

// FinishCheckItem applies an item's evidence (see ItemFinish) and returns its outcome.
// ErrCheckClosed when the session is not active or already moved past the item.
func (s *SQLiteStore) FinishCheckItem(ctx context.Context, f ItemFinish) (string, error) {
	var outcome string
	err := s.tx(ctx, func(c conn) error {
		sess, err := lockSession(ctx, c, f.UserID, f.SessionID)
		if err != nil {
			return err
		}
		if sess.Status != "active" || sess.Cursor >= len(sess.ItemIDs) || sess.ItemIDs[sess.Cursor] != f.Item.ID {
			return ErrCheckClosed
		}
		entry, grade, err := applyItemEvidence(ctx, c, f)
		if err != nil {
			return err
		}
		f.Turn.UserID, f.Turn.SessionID, f.Turn.Outcome, f.Turn.Grade = f.UserID, f.SessionID, entry.Outcome, grade
		if f.Turn.ID != "" {
			err = updateTurn(ctx, c, f.Turn)
		} else {
			_, err = insertTurn(ctx, c, f.Turn)
		}
		if err != nil {
			return err
		}
		outcome = entry.Outcome
		return advanceSession(ctx, c, f, sess.Cursor, len(sess.ItemIDs), entry)
	})
	if err != nil {
		return "", err
	}
	return outcome, nil
}

func applyItemEvidence(ctx context.Context, c conn, f ItemFinish) (ResultEntry, GradeRecord, error) {
	grade := f.Turn.Grade
	entry := ResultEntry{ItemID: f.Item.ID, ObjectiveID: f.Item.ObjectiveID, Missing: grade.Missing,
		Anchor: f.Item.Rubric.Anchor, Misconception: grade.Misconception}
	if entry.Missing == nil {
		entry.Missing = []string{}
	}
	if f.Item.ObjectiveID == "" { // 통합 문항: 상자는 움직이지 않고 covers 에 integrative_ok 만.
		entry.Outcome = learning.ItemOutcome(f.Result, nil, f.Now)
		if f.IntegrativeOK {
			return entry, grade, creditIntegrative(ctx, c, f.UserID, f.Item.Rubric.Covers)
		}
		return entry, grade, nil
	}
	m, updated, exists, err := lockMastery(ctx, c, f.UserID, f.Item.ObjectiveID)
	if err != nil {
		return entry, grade, fmt.Errorf("load mastery: %w", err)
	}
	prev := m
	entry.Outcome = learning.ItemOutcome(f.Result, m.LastFailAt, f.Now)
	entry.LevelBefore = learning.DisplayLevel(m)
	grade.Prev, grade.PrevExists = &prev, exists
	if entry.Outcome != learning.OutcomeHeld {
		m = learning.QueueEntry(prev, learning.ApplyEvidence(m, entry.Outcome, f.Now, f.Intervals), f.Hold)
		if updated, err = saveMastery(ctx, c, f.UserID, f.Item.ObjectiveID, f.Turn.Grader, m); err != nil {
			return entry, grade, err
		}
	}
	if exists || entry.Outcome != learning.OutcomeHeld {
		grade.AppliedAt = &updated
	}
	entry.LevelAfter, entry.NextReviewAt, entry.Lapses, entry.Queued = learning.DisplayLevel(m), m.NextReviewAt, m.Lapses, m.Queued
	return entry, grade, nil
}

func advanceSession(ctx context.Context, c conn, f ItemFinish, cursor, items int, entry ResultEntry) error {
	raw, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	status, next := "active", cursor+1
	switch {
	case f.Abandon:
		status, next = "abandoned", cursor
	case f.NextAsk == nil || next >= items:
		status = "finished"
	}
	// jsonb_set(result, '{items}', result->'items' || [entry]) → json_set + json_insert('$[#]').
	if _, err := c.exec(ctx, `
		UPDATE check_sessions SET cursor = $2, status = $3, updated_at = NOW(),
			result = json_set(result, '$.items', json_insert(COALESCE(json_extract(result, '$.items'), json('[]')), '$[#]', json($4))),
			finished_at = CASE WHEN $3 IN ('finished', 'abandoned') THEN NOW() ELSE finished_at END
		WHERE id = $1`, f.SessionID, next, status, string(raw)); err != nil {
		return fmt.Errorf("advance check session: %w", err)
	}
	if status != "active" {
		return nil
	}
	ask := *f.NextAsk
	ask.UserID, ask.SessionID = f.UserID, f.SessionID
	_, err = insertTurn(ctx, c, ask)
	return err
}

// DisputeResolve completes a dispute turn with the regrade (design §4.2).
type DisputeResolve struct {
	UserID      string
	SessionID   string
	ObjectiveID string
	// JudgedTurnID: the item's last judgement turn (holds grade.prev / appliedAt).
	JudgedTurnID string
	Accepted     bool
	Turn         TurnWrite
}

// ResolveDispute stores the regrade. When accepted, the item's evidence is
// rolled back to grade.prev and best rises to unaided (learning.Restore); if
// other evidence reached the objective since (updated_at differs), only best
// rises. Returns whether that conflict happened.
func (s *SQLiteStore) ResolveDispute(ctx context.Context, d DisputeResolve) (bool, error) {
	conflict := false
	err := s.tx(ctx, func(c conn) error {
		if _, err := lockSession(ctx, c, d.UserID, d.SessionID); err != nil {
			return err
		}
		if d.Accepted && d.ObjectiveID != "" {
			var err error
			if conflict, err = restoreEvidence(ctx, c, d); err != nil {
				return err
			}
		}
		accepted := d.Accepted
		d.Turn.UserID, d.Turn.Grade.Accepted, d.Turn.Grade.Conflict = d.UserID, &accepted, conflict
		return updateTurn(ctx, c, d.Turn)
	})
	if err != nil {
		return false, err
	}
	return conflict, nil
}

func restoreEvidence(ctx context.Context, c conn, d DisputeResolve) (bool, error) {
	var raw json.RawMessage
	err := c.row(ctx, `SELECT grade FROM check_turns WHERE user_id = $1 AND id = $2`, d.UserID, d.JudgedTurnID).Scan(rawJSON(&raw))
	if errors.Is(err, sql.ErrNoRows) {
		return false, ErrNotFound
	}
	if err != nil {
		return false, err
	}
	var judged GradeRecord
	if err := json.Unmarshal(raw, &judged); err != nil {
		return false, fmt.Errorf("decode judged grade: %w", err)
	}
	prev := learning.NewMastery()
	if judged.Prev != nil {
		prev = *judged.Prev
	}
	cur, updated, exists, err := lockMastery(ctx, c, d.UserID, d.ObjectiveID)
	if err != nil {
		return false, err
	}
	conflict := exists && (judged.AppliedAt == nil || !updated.Equal(*judged.AppliedAt))
	_, err = saveMastery(ctx, c, d.UserID, d.ObjectiveID, d.Turn.Grader, learning.Restore(cur, prev, conflict))
	return conflict, err
}
