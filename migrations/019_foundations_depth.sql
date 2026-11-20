-- punisher:foreign_keys=off
-- 0단계 기초 트랙(설계 docs/plans/2026-11-learner-level-design.md ②): 깊이 0 을 허용한다.
-- SQLite 는 CHECK 제약을 바꾸려면 표를 다시 만들어야 한다(007 과 같은 12단계 재생성).
-- 깊이 CHECK 가 있는 4개 표만 BETWEEN 1 AND 3 → BETWEEN 0 AND 3 으로 바꾸고, 열·기본값·인덱스는 그대로 옮긴다.
-- 기존 행은 모두 1..3 이라 변환 없음. 깊이 0 을 어느 영역에 쓰는지는 커리큘럼 검증이 정한다(foundations 만).

CREATE TABLE concepts_v19 (
    id                  TEXT PRIMARY KEY,
    name                TEXT NOT NULL,
    area                TEXT NOT NULL,
    depth               INTEGER NOT NULL CHECK (depth BETWEEN 0 AND 3),
    summary             TEXT NOT NULL DEFAULT '',
    practical_cases     TEXT NOT NULL DEFAULT '[]',
    doc_links           TEXT NOT NULL DEFAULT '[]',
    status              TEXT NOT NULL DEFAULT 'active'
                        CHECK (status IN ('active', 'proposed', 'retired')),
    origin              TEXT NOT NULL DEFAULT 'seed'
                        CHECK (origin IN ('seed', 'dynamic', 'chapter')),
    curriculum_version  TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    proposed_by         TEXT,
    owner_user_id       TEXT,
    chapter_id          TEXT REFERENCES chapters(id) ON DELETE SET NULL,
    name_en             TEXT NOT NULL DEFAULT '',
    summary_en          TEXT NOT NULL DEFAULT '',
    practical_cases_en  TEXT NOT NULL DEFAULT '[]'
);
INSERT INTO concepts_v19 (id, name, area, depth, summary, practical_cases, doc_links, status, origin,
                          curriculum_version, created_at, updated_at, proposed_by, owner_user_id, chapter_id,
                          name_en, summary_en, practical_cases_en)
SELECT id, name, area, depth, summary, practical_cases, doc_links, status, origin,
       curriculum_version, created_at, updated_at, proposed_by, owner_user_id, chapter_id,
       name_en, summary_en, practical_cases_en
FROM concepts;
DROP TABLE concepts;
ALTER TABLE concepts_v19 RENAME TO concepts;
CREATE INDEX IF NOT EXISTS idx_concepts_area_status ON concepts (area, status);
CREATE INDEX IF NOT EXISTS idx_concepts_owner ON concepts (owner_user_id) WHERE owner_user_id IS NOT NULL;

CREATE TABLE objectives_v19 (
    id          TEXT PRIMARY KEY,
    concept_id  TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    text        TEXT NOT NULL,
    depth       INTEGER NOT NULL CHECK (depth BETWEEN 0 AND 3),
    ordinal     INTEGER NOT NULL,
    status      TEXT NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'retired')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    text_en     TEXT NOT NULL DEFAULT '',
    CONSTRAINT objectives_concept_ordinal UNIQUE (concept_id, ordinal)
);
INSERT INTO objectives_v19 (id, concept_id, text, depth, ordinal, status, created_at, updated_at, text_en)
SELECT id, concept_id, text, depth, ordinal, status, created_at, updated_at, text_en
FROM objectives;
DROP TABLE objectives;
ALTER TABLE objectives_v19 RENAME TO objectives;

CREATE TABLE lectures_v19 (
    id                   TEXT PRIMARY KEY,
    concept_id           TEXT NOT NULL REFERENCES concepts(id),
    objective_ids        TEXT NOT NULL DEFAULT '[]',
    objective_key        TEXT NOT NULL,
    depth                INTEGER NOT NULL CHECK (depth BETWEEN 0 AND 3),
    kind                 TEXT NOT NULL CHECK (kind IN ('library', 'instant')),
    status               TEXT NOT NULL DEFAULT 'draft'
                         CHECK (status IN ('draft', 'needs_revision', 'published', 'archived')),
    version              INTEGER NOT NULL DEFAULT 1,
    title                TEXT NOT NULL DEFAULT '',
    content              TEXT NOT NULL DEFAULT '{}',
    content_hash         TEXT NOT NULL DEFAULT '',
    sources              TEXT NOT NULL DEFAULT '[]',
    model                TEXT NOT NULL DEFAULT '',
    prompt_version       TEXT NOT NULL DEFAULT '',
    requested_by         TEXT,
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    published_at         TEXT,
    flagged              INTEGER NOT NULL DEFAULT 0,
    flag_reason          TEXT NOT NULL DEFAULT '',
    estimated_minutes    INTEGER NOT NULL DEFAULT 0,
    generation_job_id    TEXT,
    minutes_estimated    INTEGER NOT NULL DEFAULT 0,
    review_requested_at  TEXT,
    language             TEXT NOT NULL DEFAULT 'ko',
    reader_level         TEXT NOT NULL DEFAULT 'practitioner',
    CONSTRAINT lectures_unique_version UNIQUE (concept_id, objective_key, depth, version)
);
INSERT INTO lectures_v19 (id, concept_id, objective_ids, objective_key, depth, kind, status, version, title,
                          content, content_hash, sources, model, prompt_version, requested_by, created_at,
                          published_at, flagged, flag_reason, estimated_minutes, generation_job_id,
                          minutes_estimated, review_requested_at, language, reader_level)
SELECT id, concept_id, objective_ids, objective_key, depth, kind, status, version, title,
       content, content_hash, sources, model, prompt_version, requested_by, created_at,
       published_at, flagged, flag_reason, estimated_minutes, generation_job_id,
       minutes_estimated, review_requested_at, language, reader_level
FROM lectures;
DROP TABLE lectures;
ALTER TABLE lectures_v19 RENAME TO lectures;
CREATE INDEX IF NOT EXISTS idx_lectures_concept_status ON lectures (concept_id, status);
CREATE INDEX IF NOT EXISTS idx_lectures_target
    ON lectures (concept_id, objective_key, depth, status, version DESC);
CREATE INDEX IF NOT EXISTS idx_lectures_content_hash ON lectures (content_hash);
CREATE INDEX IF NOT EXISTS lectures_target_language_idx
    ON lectures (concept_id, objective_key, depth, language, reader_level, status);

CREATE TABLE user_objective_states_v19 (
    user_id       TEXT NOT NULL,
    concept_id    TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    objective_id  TEXT NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    depth         INTEGER NOT NULL CHECK (depth BETWEEN 0 AND 3),
    state         TEXT NOT NULL CHECK (state IN ('in_progress', 'completed', 'known')),
    lecture_id    TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    updated_at    TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, concept_id, objective_id, depth)
);
INSERT INTO user_objective_states_v19 (user_id, concept_id, objective_id, depth, state, lecture_id, updated_at)
SELECT user_id, concept_id, objective_id, depth, state, lecture_id, updated_at
FROM user_objective_states;
DROP TABLE user_objective_states;
ALTER TABLE user_objective_states_v19 RENAME TO user_objective_states;
CREATE INDEX IF NOT EXISTS idx_user_objective_states_user ON user_objective_states (user_id, state);
