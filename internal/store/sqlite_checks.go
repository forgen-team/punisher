package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/forgen-team/punisher/internal/models"
)

var (
	// ErrCheckBusy: the session already has a running turn, or the user has too many running.
	ErrCheckBusy = errors.New("check turn already running")
	// ErrCheckClosed: the session is not active (finished, abandoned, failed) or the item moved on.
	ErrCheckClosed = errors.New("check session is not active")
	// errTurnSeq: another turn took the seq.
	errTurnSeq = errors.New("check turn seq conflict")
)

// constraintErr maps unique violations of the check tables (witems maps them by
// constraint name; SQLite names the columns instead): the partial index
// check_turns_one_pending is on (session_id), check_turns_seq on (session_id, seq).
func constraintErr(err error) error {
	if !isUniqueViolation(err) {
		return err
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "check_turns.seq"):
		return errTurnSeq
	case strings.Contains(msg, "check_turns.session_id"):
		return ErrCheckBusy
	case strings.Contains(msg, "check_sessions.session_key"):
		return ErrCheckBusy // 같은 키의 열린 세션이 이미 있다(다시 시도가 그 세션과 겹침).
	}
	return err
}

const sessionColumns = `id, kind, grader, COALESCE(concept_id, ''), COALESCE(lecture_id, ''), session_key,
	item_ids, cursor, status, result, error, created_at, updated_at, finished_at`

func scanSession(row rowScanner) (models.CheckSession, error) {
	var c models.CheckSession
	err := row.Scan(&c.ID, &c.Kind, &c.Grader, &c.ConceptID, &c.LectureID, &c.SessionKey,
		strs(&c.ItemIDs), &c.Cursor, &c.Status, rawJSON(&c.Result), &c.Error, ts(&c.CreatedAt), ts(&c.UpdatedAt), nts(&c.FinishedAt))
	if c.ItemIDs == nil {
		c.ItemIDs = []string{}
	}
	return c, err
}

// NewCheckSession is the insert payload of a session.
type NewCheckSession struct {
	UserID     string
	Kind       string
	Grader     string
	ConceptID  string
	LectureID  string
	SessionKey string
	// MaxPending > 0: the session's generation runs without a job row (a
	// diagnostic) — a new session counts toward the user's pending limit
	// (ErrCheckBusy); an open one of the key is still returned.
	MaxPending int
}

// CreateCheckSession inserts a preparing session; when the user already has an
// open (preparing/active) session for the key, that one is returned with created=false.
func (s *SQLiteStore) CreateCheckSession(ctx context.Context, n NewCheckSession) (models.CheckSession, bool, error) {
	var out models.CheckSession
	created := false
	err := s.tx(ctx, func(c conn) error {
		if n.MaxPending > 0 {
			open, err := openSession(ctx, c, n)
			if err == nil {
				out = open
				return nil
			}
			if !errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("load open check session: %w", err)
			}
			if err := lockUserChecks(ctx, c, n.UserID, n.MaxPending); err != nil {
				return err
			}
		}
		sess, err := scanSession(c.row(ctx, `
			INSERT INTO check_sessions (id, user_id, kind, grader, concept_id, lecture_id, session_key)
			VALUES ($1, $2, $3, $4, NULLIF($5, ''), NULLIF($6, ''), $7)
			ON CONFLICT (user_id, session_key) WHERE status IN ('preparing', 'active') DO NOTHING
			RETURNING `+sessionColumns, NewID(), n.UserID, n.Kind, n.Grader, n.ConceptID, n.LectureID, n.SessionKey))
		created = err == nil
		if errors.Is(err, sql.ErrNoRows) {
			sess, err = openSession(ctx, c, n)
		}
		if err != nil {
			return fmt.Errorf("create check session: %w", err)
		}
		out = sess
		return nil
	})
	return out, created, err
}

func openSession(ctx context.Context, c conn, n NewCheckSession) (models.CheckSession, error) {
	return scanSession(c.row(ctx, `SELECT `+sessionColumns+` FROM check_sessions
		WHERE user_id = $1 AND session_key = $2 AND status IN ('preparing', 'active')`, n.UserID, n.SessionKey))
}

// RetryCheckSession moves a failed session back to preparing ('다시 시도' of a
// diagnostic) under the user's pending limit: ErrCheckBusy when over it or when
// another open session of the key exists, ErrCheckClosed when it is not failed.
func (s *SQLiteStore) RetryCheckSession(ctx context.Context, userID, id string, maxPending int) error {
	return s.tx(ctx, func(c conn) error {
		if err := lockUserChecks(ctx, c, userID, maxPending); err != nil {
			return err
		}
		res, err := c.exec(ctx, `UPDATE check_sessions SET status = 'preparing', error = '', updated_at = NOW()
			WHERE user_id = $1 AND id = $2 AND status = 'failed'`, userID, id)
		if err != nil {
			return fmt.Errorf("retry check session: %w", constraintErr(err))
		}
		if affected(res) == 0 {
			return ErrCheckClosed
		}
		return nil
	})
}

// CheckSnapshot is a session with its items and turns read in one snapshot.
type CheckSnapshot struct {
	Session models.CheckSession
	Items   []CheckItem
	Turns   []models.CheckTurn
}

// LoadCheckSnapshot reads a session, its items and turns together: a turn and
// the cursor move committed together are seen together. witems uses a
// repeatable-read transaction; here one transaction on the single connection
// already sees no other write in between.
func (s *SQLiteStore) LoadCheckSnapshot(ctx context.Context, userID, id string) (CheckSnapshot, error) {
	var snap CheckSnapshot
	err := s.tx(ctx, func(c conn) error {
		var err error
		if snap.Session, err = lockSession(ctx, c, userID, id); err != nil {
			return err
		}
		if snap.Items, err = checkItemsByID(ctx, c, userID, snap.Session.ItemIDs); err != nil {
			return err
		}
		snap.Turns, err = listCheckTurns(ctx, c, userID, id)
		return err
	})
	return snap, err
}

// GetCheckSession loads one of the user's sessions.
func (s *SQLiteStore) GetCheckSession(ctx context.Context, userID, id string) (models.CheckSession, error) {
	return lockSession(ctx, s.c(), userID, id)
}

// lockSession loads the session inside the caller's transaction. witems locks
// the row (FOR UPDATE); the single connection already serialises transactions.
func lockSession(ctx context.Context, c conn, userID, id string) (models.CheckSession, error) {
	sess, err := scanSession(c.row(ctx, `SELECT `+sessionColumns+` FROM check_sessions
		WHERE user_id = $1 AND id = $2`, userID, id))
	if errors.Is(err, sql.ErrNoRows) {
		return sess, ErrNotFound
	}
	return sess, err
}

// ActivateCheckSession attaches the items to a preparing session and asks the
// first question. A session that is no longer preparing is returned unchanged.
func (s *SQLiteStore) ActivateCheckSession(ctx context.Context, userID, id string, itemIDs []string, ask TurnWrite) (models.CheckSession, error) {
	var out models.CheckSession
	activated := false
	err := s.tx(ctx, func(c conn) error {
		sess, err := lockSession(ctx, c, userID, id)
		if err != nil || sess.Status != "preparing" {
			out = sess
			return err
		}
		if _, err := c.exec(ctx, `UPDATE check_sessions SET item_ids = $2, cursor = 0, status = 'active', error = '',
			updated_at = NOW() WHERE id = $1`, id, itemIDs); err != nil {
			return fmt.Errorf("activate check session: %w", err)
		}
		ask.UserID, ask.SessionID = userID, id
		if _, err := insertTurn(ctx, c, ask); err != nil {
			return err
		}
		activated = true
		return nil
	})
	if err != nil || !activated {
		return out, err
	}
	return s.GetCheckSession(ctx, userID, id)
}

// SetCheckSessionStatus moves a session from one of `from` to `to`
// (failed keeps errMsg; finished/abandoned stamp finished_at). ErrCheckClosed when it was not in `from`.
func (s *SQLiteStore) SetCheckSessionStatus(ctx context.Context, userID, id string, from []string, to, errMsg string) error {
	res, err := s.c().exec(ctx, `
		UPDATE check_sessions SET status = $4, error = $5, updated_at = NOW(),
			finished_at = CASE WHEN $4 IN ('finished', 'abandoned') THEN NOW() ELSE finished_at END
		WHERE user_id = $1 AND id = $2 AND status IN (SELECT value FROM json_each($3))`, userID, id, from, to, truncateText(errMsg, 2000))
	if err != nil {
		return fmt.Errorf("set check session status: %w", err)
	}
	if affected(res) == 0 {
		return ErrCheckClosed
	}
	return nil
}

// SweepStaleChecks ends check work that stopped without finishing (restart,
// crash): pending turns older than turnAge and preparing sessions (with their
// item jobs) older than prepareAge fail; active sessions untouched for
// abandonAge are abandoned (evidence already applied stays). Returns rows changed.
func (s *SQLiteStore) SweepStaleChecks(ctx context.Context, turnAge, prepareAge, abandonAge time.Duration) (int64, error) {
	return s.SweepStaleChecksFor(ctx, "", turnAge, prepareAge, abandonAge)
}

// SweepStaleChecksFor is SweepStaleChecks limited to one user's rows ("" = every
// user, the boot/scheduler sweep) — tests sweep only their own user so another
// run on the same database keeps its pending turns.
func (s *SQLiteStore) SweepStaleChecksFor(ctx context.Context, userID string, turnAge, prepareAge, abandonAge time.Duration) (int64, error) {
	now := time.Now()
	const msg = "generation interrupted"
	const mine = ` AND ($3 = '' OR user_id = $3)`
	stmts := []struct {
		sql  string
		args []any
	}{
		{`UPDATE check_turns SET status = 'failed', error = $2 WHERE status = 'pending' AND created_at <= $1` + mine,
			[]any{now.Add(-turnAge), msg, userID}},
		{`UPDATE generation_jobs SET status = 'failed', error = $2, finished_at = NOW(), updated_at = NOW()
			WHERE kind = 'check_items' AND status IN ('pending', 'running') AND updated_at <= $1` + mine, []any{now.Add(-prepareAge), msg, userID}},
		{`UPDATE check_sessions SET status = 'failed', error = $2, updated_at = NOW()
			WHERE status = 'preparing' AND updated_at <= $1` + mine, []any{now.Add(-prepareAge), msg, userID}},
		{`UPDATE check_sessions SET status = 'abandoned', finished_at = NOW(), updated_at = NOW()
			WHERE status = 'active' AND updated_at <= $1 AND ($2 = '' OR user_id = $2)`, []any{now.Add(-abandonAge), userID}},
	}
	var total int64
	for _, st := range stmts {
		res, err := s.c().exec(ctx, st.sql, st.args...)
		if err != nil {
			return total, fmt.Errorf("sweep checks: %w", err)
		}
		total += affected(res)
	}
	return total, nil
}

// pendingChecksSQL: running tutor turns, item jobs and diagnostic generations
// (a preparing pretest / known_verify / placement session has no job row).
const pendingChecksSQL = `
	SELECT (SELECT COUNT(*) FROM check_turns WHERE user_id = $1 AND status = 'pending')
	     + (SELECT COUNT(*) FROM generation_jobs WHERE user_id = $1 AND kind = 'check_items' AND status IN ('pending', 'running'))
	     + (SELECT COUNT(*) FROM check_sessions WHERE user_id = $1 AND status = 'preparing'
	          AND kind IN ('pretest', 'known_verify', 'placement'))`

// CountPendingChecks counts the user's running tutor turns, item generations and diagnostic generations.
func (s *SQLiteStore) CountPendingChecks(ctx context.Context, userID string) (int, error) {
	var n int
	err := s.c().row(ctx, pendingChecksSQL, userID).Scan(&n)
	return n, err
}

// lockUserChecks checks the user's pending limit inside the caller's
// transaction. witems takes an advisory lock first; here the single connection
// serialises the count and the insert that follows.
func lockUserChecks(ctx context.Context, c conn, userID string, maxPending int) error {
	var pending int
	if err := c.row(ctx, pendingChecksSQL, userID).Scan(&pending); err != nil {
		return err
	}
	if pending >= maxPending {
		return ErrCheckBusy
	}
	return nil
}
