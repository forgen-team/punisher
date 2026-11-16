-- Punisher v3 (SQLite 포팅, 원본 016 — 설계 docs/plans/2026-11-punisher-learning-v3-design.md §8).
-- UUID 는 TEXT(NewID()), BOOLEAN 은 INTEGER 0/1, TIMESTAMPTZ 는 UTC TEXT. ADD COLUMN 의 IF NOT EXISTS 만 뺀다.
-- 기존 행 변환 없음: 새 열은 기본값이 지금 동작과 같다(ease 2.0 = 설정한 간격 그대로).

-- 강의 속 '직접 해 보기'·실습 시도(§2, §4). 숙달 근거가 아니라 기록·강의 신호(§6)에만 쓴다.
CREATE TABLE IF NOT EXISTS lecture_exercise_attempts (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    lecture_id   TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    anchor       TEXT NOT NULL,
    content_hash TEXT NOT NULL,
    section      TEXT NOT NULL DEFAULT '',
    kind         TEXT NOT NULL,
    answer       TEXT NOT NULL DEFAULT '',
    verdict      TEXT NOT NULL DEFAULT '',
    feedback     TEXT NOT NULL DEFAULT '',
    model        TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS lecture_exercise_attempts_user_lecture_idx
    ON lecture_exercise_attempts (user_id, lecture_id, anchor, created_at DESC);

-- 확신도(§3).
ALTER TABLE check_turns ADD COLUMN learner_confidence TEXT;
ALTER TABLE check_turns ADD COLUMN confident_error INTEGER NOT NULL DEFAULT 0;

-- 적응형 간격(§7).
ALTER TABLE user_objective_mastery ADD COLUMN ease REAL NOT NULL DEFAULT 2.0;

-- 설정: 실습 환경(macos|linux|none, '' = 이 컴퓨터의 OS), 적응형 간격 켜기.
ALTER TABLE user_learning_settings ADD COLUMN lab_os TEXT NOT NULL DEFAULT '';
ALTER TABLE user_learning_settings ADD COLUMN adaptive_intervals INTEGER NOT NULL DEFAULT 1;
