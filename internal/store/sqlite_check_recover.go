package store

import (
	"context"
	"fmt"
)

// Recorded failure of check work cut off by a restart (the check screen shows its own label with '다시 시도').
const (
	InterruptedPrepareMessage = "문항 만들기가 중단됐어요"
	InterruptedTurnMessage    = "채점이 중단됐어요"
)

// RecoveredChecks counts what RecoverInterruptedChecks failed.
type RecoveredChecks struct {
	Jobs, Sessions, Turns, Attempts int64
}

// RecoverInterruptedChecks runs once at startup, before the server takes
// requests: question generation and grading run in this process only, so any
// left preparing/pending by the previous run will never finish. They fail
// instead, so the check screen shows '다시 시도' rather than spinning forever.
// Pretest and review sessions have no retry and are left as they are (a new
// session replaces them).
func (s *SQLiteStore) RecoverInterruptedChecks(ctx context.Context) (RecoveredChecks, error) {
	var r RecoveredChecks
	err := s.tx(ctx, func(c conn) error {
		res, err := c.exec(ctx, `UPDATE generation_jobs SET status = 'failed', error = 'interrupted: server restart',
				finished_at = NOW(), updated_at = NOW()
			WHERE kind = 'check_items' AND status IN ('pending', 'running')`)
		if err != nil {
			return fmt.Errorf("recover check jobs: %w", err)
		}
		r.Jobs = affected(res)
		res, err = c.exec(ctx, `UPDATE check_sessions SET status = 'failed', error = $1, updated_at = NOW()
			WHERE status = 'preparing' AND kind IN ('check', 'known_verify', 'placement', 'synthesis')`, InterruptedPrepareMessage)
		if err != nil {
			return fmt.Errorf("recover check sessions: %w", err)
		}
		r.Sessions = affected(res)
		res, err = c.exec(ctx, `UPDATE check_turns SET status = 'failed', error = $1 WHERE status = 'pending'`, InterruptedTurnMessage)
		if err != nil {
			return fmt.Errorf("recover check turns: %w", err)
		}
		r.Turns = affected(res)
		res, err = c.exec(ctx, `UPDATE lecture_exercise_attempts SET status = 'failed' WHERE status = 'pending'`)
		if err != nil {
			return fmt.Errorf("recover exercise attempts: %w", err)
		}
		r.Attempts = affected(res)
		return nil
	})
	return r, err
}
