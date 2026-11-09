-- Punisher 초기 스키마 (SQLite 포팅, 원본: apps/punisher-server/migrations/001_init.sql)
-- 시간은 UTC TEXT 'YYYY-MM-DDTHH:MM:SS.sssZ', 배열(TEXT[])·JSONB 는 JSON TEXT, UUID 는 Go 에서 생성한다.

-- ---------------------------------------------------------------------------
-- 커리큘럼(개념 그래프)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS curriculum_versions (
    version        TEXT PRIMARY KEY,
    concept_count  INTEGER NOT NULL,
    edge_count     INTEGER NOT NULL,
    alias_count    INTEGER NOT NULL,
    applied_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE TABLE IF NOT EXISTS concepts (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    area                TEXT NOT NULL,
    depth               INTEGER NOT NULL CHECK (depth BETWEEN 1 AND 3),
    summary             TEXT NOT NULL DEFAULT '',
    practical_cases     TEXT NOT NULL DEFAULT '[]',
    doc_links           TEXT NOT NULL DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'proposed', 'retired')),
    origin              TEXT NOT NULL DEFAULT 'seed'
                        CHECK (origin IN ('seed', 'dynamic')),
    curriculum_version  TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_concepts_area_status ON concepts (area, status);

CREATE TABLE IF NOT EXISTS concept_aliases (
    alias       TEXT PRIMARY KEY,
    display     TEXT NOT NULL,
    concept_id  TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    origin      TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'dynamic')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_concept_aliases_concept ON concept_aliases (concept_id);

CREATE TABLE IF NOT EXISTS concept_edges (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    from_id     TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    to_id       TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL CHECK (kind IN ('prerequisite', 'related', 'part_of')),
    origin      TEXT NOT NULL DEFAULT 'seed' CHECK (origin IN ('seed', 'dynamic')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CONSTRAINT concept_edges_no_self CHECK (from_id <> to_id),
    CONSTRAINT concept_edges_unique UNIQUE (from_id, to_id, kind)
);
CREATE INDEX IF NOT EXISTS idx_concept_edges_to ON concept_edges (to_id, kind);

CREATE TABLE IF NOT EXISTS objectives (
    id          TEXT PRIMARY KEY,
    concept_id  TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    text        TEXT NOT NULL,
    depth       INTEGER NOT NULL CHECK (depth BETWEEN 1 AND 3),
    ordinal     INTEGER NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CONSTRAINT objectives_concept_ordinal UNIQUE (concept_id, ordinal)
);

-- ---------------------------------------------------------------------------
-- 강의
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS lectures (
    id              TEXT PRIMARY KEY,
    concept_id      TEXT NOT NULL REFERENCES concepts(id),
    objective_ids   TEXT NOT NULL DEFAULT '[]',
    objective_key   TEXT NOT NULL,
    depth           INTEGER NOT NULL CHECK (depth BETWEEN 1 AND 3),
    kind            TEXT NOT NULL CHECK (kind IN ('library', 'instant')),
    status          TEXT NOT NULL DEFAULT 'draft'
                    CHECK (status IN ('draft', 'needs_revision', 'published', 'archived')),
    version         INTEGER NOT NULL DEFAULT 1,
    title           TEXT NOT NULL DEFAULT '',
    content         TEXT NOT NULL DEFAULT '{}',
    content_hash    TEXT NOT NULL DEFAULT '',
    sources         TEXT NOT NULL DEFAULT '[]',
    model           TEXT NOT NULL DEFAULT '',
    prompt_version  TEXT NOT NULL DEFAULT '',
    requested_by    TEXT,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    published_at    TEXT,
    CONSTRAINT lectures_unique_version UNIQUE (concept_id, objective_key, depth, version)
);
CREATE INDEX IF NOT EXISTS idx_lectures_concept_status ON lectures (concept_id, status);

CREATE TABLE IF NOT EXISTS lecture_reviews (
    id          TEXT PRIMARY KEY,
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    verdict     TEXT NOT NULL CHECK (verdict IN ('pass', 'fail')),
    issues      TEXT NOT NULL DEFAULT '[]',
    model       TEXT NOT NULL DEFAULT '',
    reviewer    TEXT NOT NULL DEFAULT 'llm',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_lecture_reviews_lecture ON lecture_reviews (lecture_id);

-- ---------------------------------------------------------------------------
-- 사용자 학습 상태
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS user_concept_states (
    user_id       TEXT NOT NULL,
    concept_id    TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    state         TEXT NOT NULL
                  CHECK (state IN ('not_started', 'in_progress', 'completed', 'known')),
    completed_at  TEXT,
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, concept_id)
);
CREATE INDEX IF NOT EXISTS idx_user_concept_states_user_state ON user_concept_states (user_id, state);

CREATE TABLE IF NOT EXISTS user_objective_states (
    user_id       TEXT NOT NULL,
    concept_id    TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    objective_id  TEXT NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    depth         INTEGER NOT NULL CHECK (depth BETWEEN 1 AND 3),
    state         TEXT NOT NULL CHECK (state IN ('in_progress', 'completed', 'known')),
    lecture_id    TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, concept_id, objective_id, depth)
);
CREATE INDEX IF NOT EXISTS idx_user_objective_states_user ON user_objective_states (user_id, state);

CREATE TABLE IF NOT EXISTS user_lecture_progress (
    user_id       TEXT NOT NULL,
    lecture_id    TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    anchor        TEXT NOT NULL DEFAULT '',
    percent       INTEGER NOT NULL DEFAULT 0 CHECK (percent BETWEEN 0 AND 100),
    started_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    last_seen_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    completed_at  TEXT,
    PRIMARY KEY (user_id, lecture_id)
);
CREATE INDEX IF NOT EXISTS idx_user_lecture_progress_recent ON user_lecture_progress (user_id, last_seen_at DESC);

CREATE TABLE IF NOT EXISTS lecture_questions (
    id           TEXT PRIMARY KEY,
    user_id      TEXT NOT NULL,
    lecture_id   TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    concept_id   TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    anchor       TEXT NOT NULL DEFAULT '',
    question     TEXT NOT NULL,
    answer       TEXT NOT NULL DEFAULT '',
    status       TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'failed')),
    model        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    answered_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_lecture_questions_user_lecture ON lecture_questions (user_id, lecture_id, created_at);

CREATE TABLE IF NOT EXISTS notes (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    lecture_id  TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    concept_id  TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    anchor      TEXT NOT NULL DEFAULT '',
    kind        TEXT NOT NULL DEFAULT 'note' CHECK (kind IN ('note', 'highlight')),
    quote       TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_notes_user_lecture ON notes (user_id, lecture_id);
CREATE INDEX IF NOT EXISTS idx_notes_user_concept ON notes (user_id, concept_id);

CREATE TABLE IF NOT EXISTS lecture_feedback (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    feedback    TEXT NOT NULL
                CHECK (feedback IN ('understood', 'insufficient', 'too_easy', 'too_hard', 'want_more')),
    anchor      TEXT NOT NULL DEFAULT '',
    comment     TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_lecture_feedback_user ON lecture_feedback (user_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- 학습 요청 · 목표 · 생성 잡
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS study_requests (
    id                    TEXT PRIMARY KEY,
    user_id               TEXT NOT NULL,
    source                TEXT NOT NULL CHECK (source IN ('rss', 'manual', 'guardian')),
    source_ref            TEXT NOT NULL DEFAULT '',
    query                 TEXT NOT NULL,
    context               TEXT NOT NULL DEFAULT '{}',
    resolved_concept_ids  TEXT NOT NULL DEFAULT '[]',
    status                TEXT NOT NULL DEFAULT 'unresolved'
                          CHECK (status IN ('unresolved', 'resolved', 'in_progress', 'done', 'dismissed')),
    idempotency_key       TEXT NOT NULL,
    created_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at            TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CONSTRAINT study_requests_idempotency UNIQUE (user_id, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_study_requests_user_status ON study_requests (user_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS user_goals (
    id                  TEXT PRIMARY KEY,
    user_id             TEXT NOT NULL,
    title               TEXT NOT NULL,
    target_concept_ids  TEXT NOT NULL DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'achieved', 'archived')),
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_user_goals_user_status ON user_goals (user_id, status);

CREATE TABLE IF NOT EXISTS generation_jobs (
    id               TEXT PRIMARY KEY,
    idempotency_key  TEXT NOT NULL UNIQUE,
    kind             TEXT NOT NULL,
    user_id          TEXT,
    concept_id       TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    objective_key    TEXT NOT NULL DEFAULT '',
    depth            INTEGER,
    status           TEXT NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending', 'running', 'succeeded', 'failed', 'skipped')),
    reason           TEXT NOT NULL DEFAULT '',
    attempts         INTEGER NOT NULL DEFAULT 0,
    lecture_id       TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    model            TEXT NOT NULL DEFAULT '',
    input_tokens     INTEGER NOT NULL DEFAULT 0,
    output_tokens    INTEGER NOT NULL DEFAULT 0,
    cost_usd         REAL NOT NULL DEFAULT 0,
    started_at       TEXT,
    finished_at      TEXT,
    created_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at       TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_status ON generation_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS idx_generation_jobs_user ON generation_jobs (user_id, created_at DESC);
