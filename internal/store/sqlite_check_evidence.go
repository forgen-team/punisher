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
	Hold bool
	// Record: a pretest / placement answer — outcome recorded, no evidence (design §5 P3).
	Record bool
	// KnownVerify: the item verifies '이미 앎' (a known_verify session) —
	// learning.ApplyKnownVerify instead of ApplyEvidence. A known_verify item of a
	// review counts as one while its objective is in the verification queue.
	KnownVerify bool
	Intervals   []int
	Now         time.Time
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
	// Relearn: a '이미 앎' verification that did not pass — '다시 배우기' is offered.
	Relearn bool `json:"relearn,omitempty"`
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
		if err := otherPendingTurn(ctx, c, f.SessionID, f.Turn.ID); err != nil {
			return err
		}
		// 새 판정 턴의 id 를 먼저 정해 숙달의 근거 표지(last_evidence_turn_id)로 쓴다
		// (witems 는 gen_random_uuid() 로 미리 발급, 여기선 NewID()).
		insert := f.Turn.ID == ""
		if insert {
			f.Turn.ID = NewID()
		}
		entry, grade, err := applyItemEvidence(ctx, c, f)
		if err != nil {
			return err
		}
		f.Turn.UserID, f.Turn.SessionID, f.Turn.Outcome, f.Turn.Grade = f.UserID, f.SessionID, entry.Outcome, grade
		if insert {
			_, err = insertTurn(ctx, c, f.Turn)
		} else {
			err = updateTurn(ctx, c, f.Turn)
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
	if f.Record {
		entry.Outcome = learning.OutcomeRecorded
		return entry, grade, nil
	}
	if f.Item.ObjectiveID == "" { // 통합 문항: 상자는 움직이지 않고 covers 에 integrative_ok 만.
		entry.Outcome = learning.ItemOutcome(f.Result, nil, f.Now)
		if f.IntegrativeOK {
			return entry, grade, creditIntegrative(ctx, c, f.UserID, f.Item.Rubric.Covers)
		}
		return entry, grade, nil
	}
	l, err := lockMastery(ctx, c, f.UserID, f.Item.ObjectiveID)
	if err != nil {
		return entry, grade, fmt.Errorf("load mastery: %w", err)
	}
	m, updated, mark, exists := l.M, l.Updated, l.Mark, l.Exists
	prev := m
	entry.Outcome = learning.ItemOutcome(f.Result, m.LastFailAt, f.Now)
	entry.LevelBefore = learning.DisplayLevel(m)
	grade.Prev, grade.PrevExists = &prev, exists
	verify := f.KnownVerify || (f.Item.Purpose == check.PurposeKnownVerify && m.Verify)
	entry.Relearn = verify && learning.SuggestRelearn(entry.Outcome)
	if entry.Outcome != learning.OutcomeHeld {
		next := learning.ApplyEvidence(m, entry.Outcome, f.Now, f.Intervals)
		next.Verify = false // 판정된 근거가 생기면 '이미 앎' 확인 대기열에서 빠진다.
		if verify {
			next = learning.ApplyKnownVerify(m, entry.Outcome, f.Now, f.Intervals)
		}
		m = learning.QueueEntry(prev, next, f.Hold)
		if updated, err = saveMastery(ctx, c, f.UserID, f.Item.ObjectiveID, f.Turn.Grader, m, f.Turn.ID); err != nil {
			return entry, grade, err
		}
		mark = f.Turn.ID
	}
	grade.Mark = &mark
	if entry.Outcome == learning.OutcomeUnaidedCorrect { // 혼자 맞히면 그 목표의 인정한 오개념은 고친 것(design §5 P4).
		if err := resolveMisconceptions(ctx, c, f.UserID, f.Item.ObjectiveID); err != nil {
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
	// JudgedTurnID: the item's last judgement turn (holds grade.prev / mark).
	JudgedTurnID string
	Accepted     bool
	Turn         TurnWrite
	// ConflictText replaces Turn.Text when the dispute is accepted but other
	// evidence came since (the box is kept).
	ConflictText string
	Now          time.Time
	Intervals    []int
}

// ResolveDispute stores the regrade. When accepted, the item's evidence is
// rolled back to grade.prev and best rises to unaided (learning.Restore,
// keeping what changed since without being evidence — learning.KeepSince); if
// other evidence reached the objective since (the last-evidence marker moved),
// only best rises. Returns whether that conflict happened.
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
		if conflict && d.ConflictText != "" {
			d.Turn.Text = d.ConflictText
		}
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
	l, err := lockMastery(ctx, c, d.UserID, d.ObjectiveID)
	if err != nil {
		return false, err
	}
	// 충돌 = 그 판정 뒤에 다른 근거가 반영됐거나 다시 배우기로 초기화됨(표지가 바뀜). 통합 크레딧·대기 해제·stale 은 표지를 건드리지 않는다.
	conflict := l.Exists && (judged.AppliedAt == nil || !l.Updated.Equal(*judged.AppliedAt))
	if judged.Mark != nil {
		conflict = l.Exists && l.Mark != *judged.Mark
	}
	restored := learning.Restore(l.M, prev, conflict)
	if !conflict {
		restored = learning.KeepSince(restored, l.M, d.Now, d.Intervals)
	}
	restored.Verify = false // 받아들여진 이의 = 맞은 답: '이미 앎' 확인도 통과로 끝난다.
	_, err = saveMastery(ctx, c, d.UserID, d.ObjectiveID, d.Turn.Grader, restored, d.Turn.ID)
	return conflict, err
}
