package store

import (
	"context"
	"testing"
)

// 재시작 전에 돌던 문항 만들기·채점은 실패로 바뀌어 '다시 시도'로 이어 갈 수 있고, 사전 테스트·진행 중 세션은 그대로 둔다.
func TestRecoverInterruptedChecks(t *testing.T) {
	s, _ := newTestStore(t)
	ctx := context.Background()
	exec := func(q string, a ...any) {
		if _, err := s.c().exec(ctx, q, a...); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
	}
	exec(`INSERT INTO check_sessions (id, user_id, kind, session_key, status) VALUES ('s-check', $1, 'check', 'k1', 'preparing')`, testUser)
	exec(`INSERT INTO check_sessions (id, user_id, kind, session_key, status) VALUES ('s-pre', $1, 'pretest', 'k2', 'preparing')`, testUser)
	exec(`INSERT INTO check_sessions (id, user_id, kind, session_key, status) VALUES ('s-active', $1, 'known_verify', 'k3', 'active')`, testUser)
	exec(`INSERT INTO check_turns (id, session_id, user_id, seq, role, move, status) VALUES ('t1', 's-active', $1, 1, 'tutor', 'ask', 'pending')`, testUser)
	exec(`INSERT INTO generation_jobs (id, idempotency_key, kind, user_id, objective_key, status, attempts, started_at)
		VALUES ('j1', 'k', 'check_items', $1, 'k', 'running', 1, NOW())`, testUser)

	r, err := s.RecoverInterruptedChecks(ctx)
	if err != nil {
		t.Fatalf("recover: %v", err)
	}
	if r != (RecoveredChecks{Jobs: 1, Sessions: 1, Turns: 1}) {
		t.Fatalf("recovered = %+v", r)
	}
	status := func(q string) (st, msg string) {
		if err := s.c().row(ctx, q).Scan(&st, &msg); err != nil {
			t.Fatalf("%s: %v", q, err)
		}
		return st, msg
	}
	if st, msg := status(`SELECT status, error FROM check_sessions WHERE id = 's-check'`); st != "failed" || msg != InterruptedPrepareMessage {
		t.Fatalf("preparing check = %s %q", st, msg)
	}
	if st, _ := status(`SELECT status, error FROM check_sessions WHERE id = 's-pre'`); st != "preparing" {
		t.Fatalf("pretest changed to %s", st)
	}
	if st, _ := status(`SELECT status, error FROM check_sessions WHERE id = 's-active'`); st != "active" {
		t.Fatalf("active changed to %s", st)
	}
	if st, msg := status(`SELECT status, error FROM check_turns WHERE id = 't1'`); st != "failed" || msg != InterruptedTurnMessage {
		t.Fatalf("turn = %s %q", st, msg)
	}
	if st, _ := status(`SELECT status, error FROM generation_jobs WHERE id = 'j1'`); st != "failed" {
		t.Fatalf("job = %s", st)
	}
}
