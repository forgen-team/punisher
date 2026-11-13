-- 개인화(P4) (SQLite 포팅, 원본 013 — 설계 docs/plans/2026-11-punisher-mastery-design.md §3.1 · §5 P4).
-- 사용자가 확인 결과에서 인정한 오개념만 저장한다(텍스트는 그 턴의 grade.misconception).
-- status(open | resolved)는 CHECK 없이 Go 에서 검증한다(§0.3). UUID 는 Go 에서 store.NewID(), TIMESTAMPTZ 는 UTC TEXT.

CREATE TABLE IF NOT EXISTS user_misconceptions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    concept_id   TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    objective_id TEXT REFERENCES objectives(id) ON DELETE CASCADE,  -- NULL = 통합 문항
    text         TEXT NOT NULL,
    turn_id      TEXT REFERENCES check_turns(id) ON DELETE SET NULL,
    status       TEXT NOT NULL DEFAULT 'open',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    resolved_at  TEXT
);
-- 같은 턴의 오개념은 한 번만 인정된다.
CREATE UNIQUE INDEX IF NOT EXISTS user_misconceptions_turn ON user_misconceptions (user_id, turn_id);
CREATE INDEX IF NOT EXISTS user_misconceptions_user_idx ON user_misconceptions (user_id, status, concept_id);
