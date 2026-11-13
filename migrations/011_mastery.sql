-- 숙달 확인(P1)·간격 복습(P2) (SQLite 포팅, 원본 011 — 설계 docs/plans/2026-11-punisher-mastery-design.md §3.2).
-- 열거값(kind·purpose·format·move·verdict·grader·outcome·best)은 Go 에서 검증한다(CHECK 없음, 원본과 같음).
-- UUID 는 Go 에서 store.NewID(), JSONB 는 JSON TEXT, 부분 유니크 인덱스는 그대로.
-- 문항 은행. 공개: 개념 visibleTo AND (lecture_id IS NULL OR 강의가 library 이거나 requested_by=user)
--             AND (prompt_hash = '' OR created_for = user).
CREATE TABLE IF NOT EXISTS check_items (
    id               TEXT PRIMARY KEY,
    concept_id       TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    objective_id     TEXT REFERENCES objectives(id) ON DELETE CASCADE,  -- NULL = 통합 문항
    lecture_id       TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    purpose          TEXT NOT NULL,    -- check | review (P3: pretest | known_verify | placement)
    format           TEXT NOT NULL,    -- recall | predict | apply | integrative (P3: choice)
    prompt           TEXT NOT NULL,
    material         TEXT NOT NULL DEFAULT '{}',
    rubric           TEXT NOT NULL DEFAULT '{}',   -- {must, nice, misconceptions, answer, anchor, covers[]}
    variant_of       TEXT REFERENCES check_items(id) ON DELETE SET NULL,
    objective_text   TEXT NOT NULL DEFAULT '',
    objective_depth  INTEGER NOT NULL DEFAULT 0,
    prompt_hash      TEXT NOT NULL DEFAULT '',
    created_for      TEXT,
    generation_job_id TEXT,
    status           TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    model            TEXT NOT NULL DEFAULT '',
    prompt_version   TEXT NOT NULL DEFAULT '',
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS check_items_objective_idx ON check_items (objective_id, status);
CREATE INDEX IF NOT EXISTS check_items_lecture_idx ON check_items (lecture_id, purpose, prompt_hash);

CREATE TABLE IF NOT EXISTS check_sessions (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    kind        TEXT NOT NULL,          -- check | review (P3: pretest | known_verify | placement)
    grader      TEXT NOT NULL DEFAULT 'llm',  -- 세션 생성 시 고정(llm | self)
    concept_id  TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    lecture_id  TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    session_key TEXT NOT NULL,          -- 'lecture:<id>' | 'review:<KST 날짜>'
    item_ids    TEXT NOT NULL DEFAULT '[]',
    cursor      INTEGER NOT NULL DEFAULT 0,
    status      TEXT NOT NULL DEFAULT 'preparing'
                CHECK (status IN ('preparing', 'active', 'finished', 'abandoned', 'failed')),
    result      TEXT NOT NULL DEFAULT '{}',
    error       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    finished_at TEXT
);
CREATE INDEX IF NOT EXISTS check_sessions_user_idx ON check_sessions (user_id, created_at DESC);
CREATE UNIQUE INDEX IF NOT EXISTS check_sessions_one_open
    ON check_sessions (user_id, session_key) WHERE status IN ('preparing', 'active');

CREATE TABLE IF NOT EXISTS check_turns (
    id          TEXT PRIMARY KEY,
    session_id  TEXT NOT NULL REFERENCES check_sessions(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    item_id     TEXT REFERENCES check_items(id) ON DELETE SET NULL,
    seq         INTEGER NOT NULL,
    role        TEXT NOT NULL,          -- tutor | learner
    move        TEXT NOT NULL DEFAULT '',  -- ask|narrow|explain|reask|challenge|wrap|answer|reveal|self_grade|dispute
    text        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'ready' CHECK (status IN ('pending', 'ready', 'failed')),
    verdict     TEXT,                   -- correct|partial|incorrect|revealed|held
    outcome     TEXT,                   -- 문항 마지막 턴에만: unaided_correct|unaided_partial|aided|failed|held
    aided       INTEGER NOT NULL DEFAULT 0,
    grader      TEXT NOT NULL DEFAULT '',  -- llm | self | llm_regrade
    reused_item INTEGER NOT NULL DEFAULT 0,
    grade       TEXT NOT NULL DEFAULT '{}',  -- {met, missing, quote, misconception, confidence, prev}
    model       TEXT NOT NULL DEFAULT '',
    input_tokens  INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    error       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CONSTRAINT check_turns_seq UNIQUE (session_id, seq)
);
CREATE UNIQUE INDEX IF NOT EXISTS check_turns_one_pending ON check_turns (session_id) WHERE status = 'pending';

CREATE TABLE IF NOT EXISTS user_objective_mastery (
    user_id          TEXT NOT NULL,
    objective_id     TEXT NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    concept_id       TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    box              INTEGER NOT NULL DEFAULT 0,
    grad_step        INTEGER NOT NULL DEFAULT 0,   -- 0 사다리 중, 1 졸업 확인 대기, 2 종료
    best             TEXT NOT NULL DEFAULT 'none',
    integrative_ok   INTEGER NOT NULL DEFAULT 0,
    lapses           INTEGER NOT NULL DEFAULT 0,
    next_review_at   TEXT,
    queued           INTEGER NOT NULL DEFAULT 0,
    last_credit_at   TEXT,
    last_fail_at     TEXT,
    objective_text   TEXT NOT NULL DEFAULT '',
    objective_depth  INTEGER NOT NULL DEFAULT 0,
    stale            INTEGER NOT NULL DEFAULT 0,
    last_grader      TEXT NOT NULL DEFAULT '',
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, objective_id)
);
CREATE INDEX IF NOT EXISTS user_objective_mastery_due_idx ON user_objective_mastery (user_id, next_review_at);

ALTER TABLE user_learning_settings ADD COLUMN mastery_target   INTEGER NOT NULL DEFAULT 4;
ALTER TABLE user_learning_settings ADD COLUMN review_intervals TEXT NOT NULL DEFAULT '[1,7,30]';
ALTER TABLE user_learning_settings ADD COLUMN review_minutes   INTEGER NOT NULL DEFAULT 10;
ALTER TABLE user_learning_settings ADD COLUMN review_grader    TEXT NOT NULL DEFAULT 'self';
