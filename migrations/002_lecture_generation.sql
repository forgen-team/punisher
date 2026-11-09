-- 강의 생성 파이프라인 · 학습 API (SQLite 포팅, 원본 002_lecture_generation.sql)
-- 마이그레이션은 schema_migrations 로 한 번만 실행되므로 ADD COLUMN IF NOT EXISTS 가 필요 없다.

ALTER TABLE lectures ADD COLUMN flagged INTEGER NOT NULL DEFAULT 0;
ALTER TABLE lectures ADD COLUMN flag_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE lectures ADD COLUMN estimated_minutes INTEGER NOT NULL DEFAULT 0;
ALTER TABLE lectures ADD COLUMN generation_job_id TEXT;
CREATE INDEX IF NOT EXISTS idx_lectures_target
    ON lectures (concept_id, objective_key, depth, status, version DESC);
CREATE INDEX IF NOT EXISTS idx_lectures_content_hash ON lectures (content_hash);

ALTER TABLE lecture_reviews ADD COLUMN attempt INTEGER NOT NULL DEFAULT 1;
ALTER TABLE lecture_reviews ADD COLUMN summary TEXT NOT NULL DEFAULT '';
ALTER TABLE lecture_reviews ADD COLUMN validation_issues TEXT NOT NULL DEFAULT '[]';

-- 같은 목표 집합·깊이의 동시 생성은 활성 잡 부분 유일 인덱스로 차단.
ALTER TABLE generation_jobs ADD COLUMN input TEXT NOT NULL DEFAULT '{}';
ALTER TABLE generation_jobs ADD COLUMN error TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS generation_jobs_one_active_target
    ON generation_jobs (objective_key, depth)
    WHERE status IN ('pending', 'running') AND kind IN ('library', 'instant');
CREATE INDEX IF NOT EXISTS idx_generation_jobs_budget ON generation_jobs (user_id, kind, created_at);

-- 질문 스레드: 다른 방식 재설명(alt_explanation)도 같은 테이블에 저장.
ALTER TABLE lecture_questions ADD COLUMN kind TEXT NOT NULL DEFAULT 'question'
    CHECK (kind IN ('question', 'alt_explanation'));
ALTER TABLE lecture_questions ADD COLUMN linked_concept_ids TEXT NOT NULL DEFAULT '[]';
ALTER TABLE lecture_questions ADD COLUMN proposed_concepts TEXT NOT NULL DEFAULT '[]';
ALTER TABLE lecture_questions ADD COLUMN error TEXT NOT NULL DEFAULT '';

ALTER TABLE lecture_feedback ADD COLUMN alt_request_id TEXT REFERENCES lecture_questions(id) ON DELETE SET NULL;

-- 용어 팝오버 캐시(개념당 1개, haiku 생성).
CREATE TABLE IF NOT EXISTS term_glosses (
    concept_id  TEXT PRIMARY KEY REFERENCES concepts(id) ON DELETE CASCADE,
    gloss       TEXT NOT NULL,
    model       TEXT NOT NULL DEFAULT '',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- 학습 요청 해석 워커 상태.
ALTER TABLE study_requests ADD COLUMN starting_concept_id TEXT REFERENCES concepts(id) ON DELETE SET NULL;
ALTER TABLE study_requests ADD COLUMN resolve_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE study_requests ADD COLUMN resolve_error TEXT NOT NULL DEFAULT '';
ALTER TABLE study_requests ADD COLUMN resolved_at TEXT;

-- 즉석 제안 개념의 제안자.
ALTER TABLE concepts ADD COLUMN proposed_by TEXT;
