package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/wooo-jin/punisher/internal/learning"
	"github.com/wooo-jin/punisher/internal/models"
)

// TurnWrite is a turn to insert, or (ID set) the fields that complete a pending turn.
type TurnWrite struct {
	ID           string
	UserID       string
	SessionID    string
	ItemID       string
	Role         string
	Move         string
	Text         string
	Status       string // "" = ready
	Verdict      string
	Outcome      string
	Aided        bool
	Grader       string
	ReusedItem   bool
	Grade        GradeRecord
	Model        string
	InputTokens  int
	OutputTokens int
	// Confidence: learner turn only (v3 §3); ConfidentError: the item's last judgement.
	Confidence     string
	ConfidentError bool
}

// GradeRecord is check_turns.grade: the evidence of a judgement plus, on the
// item's last judgement, the mastery before it was applied (Prev) and the
// objective's last-evidence marker right after (Mark: this turn when its
// evidence was saved, else the marker it found; "" = none) — what a dispute
// restores and how it detects later evidence (design §2.3, §4.2). AppliedAt
// (updated_at right after) is the marker of turns written before Mark existed.
type GradeRecord struct {
	models.CheckGrade
	Prev       *learning.Mastery `json:"prev,omitempty"`
	PrevExists bool              `json:"prevExists,omitempty"`
	AppliedAt  *time.Time        `json:"appliedAt,omitempty"`
	Mark       *string           `json:"mark,omitempty"`
}

const turnColumns = `id, COALESCE(item_id, ''), seq, role, move, text, status, COALESCE(verdict, ''),
	COALESCE(outcome, ''), aided, grader, reused_item, grade, model, error, created_at,
	COALESCE(learner_confidence, ''), confident_error`

func scanTurn(row rowScanner) (models.CheckTurn, error) {
	var t models.CheckTurn
	err := row.Scan(&t.ID, &t.ItemID, &t.Seq, &t.Role, &t.Move, &t.Text, &t.Status, &t.Verdict,
		&t.Outcome, &t.Aided, &t.Grader, &t.ReusedItem, rawJSON(&t.GradeRaw), &t.Model, &t.Error, ts(&t.CreatedAt),
		&t.Confidence, &t.ConfidentError)
	return t, err
}

// ListCheckTurns returns a session's turns in order.
func (s *SQLiteStore) ListCheckTurns(ctx context.Context, userID, sessionID string) ([]models.CheckTurn, error) {
	return listCheckTurns(ctx, s.c(), userID, sessionID)
}

func listCheckTurns(ctx context.Context, c conn, userID, sessionID string) ([]models.CheckTurn, error) {
	list, err := collect(c.query(ctx, `SELECT `+turnColumns+` FROM check_turns
		WHERE user_id = $1 AND session_id = $2 ORDER BY seq`, userID, sessionID), scanTurn)
	if err != nil {
		return nil, fmt.Errorf("list check turns: %w", err)
	}
	return list, nil
}

func gradeJSON(g GradeRecord) (string, error) {
	if g.Met == nil {
		g.Met = []string{}
	}
	if g.Missing == nil {
		g.Missing = []string{}
	}
	raw, err := json.Marshal(g)
	return string(raw), err
}

// insertTurn appends a turn with the next seq (t.ID = a pre-issued id, ""
// = a new one). witems retries a seq taken concurrently in a savepoint; here
// the transaction runs alone on the single connection, so the seq cannot be
// taken and the one-pending index is the only conflict (ErrCheckBusy).
func insertTurn(ctx context.Context, c conn, t TurnWrite) (models.CheckTurn, error) {
	grade, err := gradeJSON(t.Grade)
	if err != nil {
		return models.CheckTurn{}, err
	}
	if t.Status == "" {
		t.Status = "ready"
	}
	if t.ID == "" {
		t.ID = NewID()
	}
	saved, err := scanTurn(c.row(ctx, `
		INSERT INTO check_turns (id, session_id, user_id, item_id, seq, role, move, text, status, verdict, outcome,
			aided, grader, reused_item, grade, model, input_tokens, output_tokens, learner_confidence, confident_error)
		SELECT $17, $1, $2, NULLIF($3, ''),
			(SELECT COALESCE(MAX(seq), 0) + 1 FROM check_turns WHERE session_id = $1),
			$4, $5, $6, $7, NULLIF($8, ''), NULLIF($9, ''), $10, $11, $12, $13, $14, $15, $16, NULLIF($18, ''), $19
		RETURNING `+turnColumns,
		t.SessionID, t.UserID, t.ItemID, t.Role, t.Move, t.Text, t.Status, t.Verdict, t.Outcome,
		t.Aided, t.Grader, t.ReusedItem, grade, t.Model, t.InputTokens, t.OutputTokens, t.ID, t.Confidence, t.ConfidentError))
	if err != nil {
		return saved, constraintErr(err)
	}
	return saved, nil
}

// updateTurn completes a turn (pending → ready) or rewrites a ready turn's judgement.
func updateTurn(ctx context.Context, c conn, t TurnWrite) error {
	grade, err := gradeJSON(t.Grade)
	if err != nil {
		return err
	}
	res, err := c.exec(ctx, `
		UPDATE check_turns SET status = 'ready', move = $3, text = $4, verdict = NULLIF($5, ''), outcome = NULLIF($6, ''),
			aided = $7, grader = $8, grade = $9, model = $10, input_tokens = input_tokens + $11, output_tokens = output_tokens + $12, error = '',
			reused_item = (reused_item OR $13), confident_error = $14
		WHERE user_id = $1 AND id = $2 AND status IN ('pending', 'ready')`,
		t.UserID, t.ID, t.Move, t.Text, t.Verdict, t.Outcome, t.Aided, t.Grader, grade, t.Model, t.InputTokens, t.OutputTokens, t.ReusedItem,
		t.ConfidentError)
	if err != nil {
		return fmt.Errorf("update check turn: %w", err)
	}
	if affected(res) == 0 {
		return ErrNotFound
	}
	return nil
}

// StartCheckTurns adds a learner turn (optional) and a pending tutor turn in
// one transaction (design §4.3), under the user's pending limit and the
// session's single-pending index (ErrCheckBusy). requireActive: the session
// must be active (answers); disputes may follow a finished session.
func (s *SQLiteStore) StartCheckTurns(ctx context.Context, userID, sessionID string, learner *TurnWrite, tutor *TurnWrite, maxPending int, requireActive bool) (models.CheckTurn, error) {
	var last models.CheckTurn
	err := s.tx(ctx, func(c conn) error {
		if tutor != nil {
			if err := lockUserChecks(ctx, c, userID, maxPending); err != nil {
				return err
			}
		}
		sess, err := lockSession(ctx, c, userID, sessionID)
		if err != nil {
			return err
		}
		if requireActive {
			if err := checkCurrentItem(ctx, c, sess, learner, tutor); err != nil {
				return err
			}
		}
		for _, t := range []*TurnWrite{learner, tutor} {
			if t == nil {
				continue
			}
			t.UserID, t.SessionID = userID, sessionID
			if last, err = insertTurn(ctx, c, *t); err != nil {
				return err
			}
		}
		_, err = c.exec(ctx, `UPDATE check_sessions SET updated_at = NOW() WHERE id = $1`, sessionID)
		return err
	})
	return last, err
}

// checkCurrentItem: an answer (or its retry) belongs to the item under the
// cursor of an active session — a reveal or self-grade that closed the item in
// between makes it ErrCheckClosed. A self-graded item takes one answer only
// (checked here, in the session's transaction, so a double submit cannot add a second).
func checkCurrentItem(ctx context.Context, c conn, sess models.CheckSession, learner, tutor *TurnWrite) error {
	if sess.Status != "active" || sess.Cursor >= len(sess.ItemIDs) {
		return ErrCheckClosed
	}
	for _, t := range []*TurnWrite{learner, tutor} {
		if t != nil && t.ItemID != sess.ItemIDs[sess.Cursor] {
			return ErrCheckClosed
		}
	}
	if learner == nil || tutor != nil || sess.Grader != learning.GraderSelf {
		return nil
	}
	var answered bool
	if err := c.row(ctx, `SELECT EXISTS (SELECT 1 FROM check_turns WHERE session_id = $1 AND item_id = $2
		AND role = 'learner' AND move = 'answer' AND status <> 'failed')`, sess.ID, learner.ItemID).Scan(&answered); err != nil {
		return fmt.Errorf("check answered: %w", err)
	}
	if answered {
		return ErrCheckClosed
	}
	return nil
}

// otherPendingTurn: another turn of the session is still running (ErrCheckBusy)
// — an item is never closed while its answer is being graded. exceptID is the
// turn being completed ("" = none). IS DISTINCT FROM NULLIF(…) → `id <> $2`
// (id is never NULL, and "" matches no id).
func otherPendingTurn(ctx context.Context, c conn, sessionID, exceptID string) error {
	var busy bool
	if err := c.row(ctx, `SELECT EXISTS (SELECT 1 FROM check_turns WHERE session_id = $1 AND status = 'pending'
		AND id <> $2)`, sessionID, exceptID).Scan(&busy); err != nil {
		return fmt.Errorf("check pending turns: %w", err)
	}
	if busy {
		return ErrCheckBusy
	}
	return nil
}

// CompleteCheckTurn stores a tutor reply that does not end the item (narrow, explain, challenge).
func (s *SQLiteStore) CompleteCheckTurn(ctx context.Context, t TurnWrite) error {
	return s.tx(ctx, func(c conn) error { return updateTurn(ctx, c, t) })
}

// FailCheckTurn records a failed tutor turn (never stored as a judgement).
func (s *SQLiteStore) FailCheckTurn(ctx context.Context, userID, turnID, model, errMsg string) error {
	_, err := s.c().exec(ctx, `UPDATE check_turns SET status = 'failed', model = $3, error = $4
		WHERE user_id = $1 AND id = $2 AND status = 'pending'`, userID, turnID, model, truncateText(errMsg, 2000))
	if err != nil {
		return fmt.Errorf("fail check turn: %w", err)
	}
	return nil
}
