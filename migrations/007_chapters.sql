-- punisher:foreign_keys=off
-- 다음 챕터 (SQLite 포팅, 원본 007): 시드 지도를 다 채운 뒤 사용자가 고르거나 입력한 주제로 개념 묶음(챕터)을 연다.
-- 챕터 개념은 그 사용자의 지도에만 보인다(owner_user_id). 공용 시드(owner_user_id IS NULL)는 건드리지 않는다.
-- concepts.origin 의 CHECK 에 'chapter' 를 더해야 하는데 SQLite 는 제약을 바꿀 수 없어 표를 다시 만든다
-- (첫 줄 표시로 외래 키를 끄고 실행 → 끝에 foreign_key_check. store/migrations.go).

CREATE TABLE IF NOT EXISTS chapters (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    title       TEXT NOT NULL,
    summary     TEXT NOT NULL DEFAULT '',
    -- 고른 제안·직접 입력(근거). 해석(LLM 산출물)은 draft 에 따로 남는다.
    input       TEXT NOT NULL DEFAULT '{}',
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_chapters_user ON chapters (user_id, created_at DESC);

-- 미리보기 초안: 검증 통과본을 저장해 두고 '이대로 열기'에서 이 초안만 반영한다.
CREATE TABLE IF NOT EXISTS chapter_drafts (
    id          TEXT PRIMARY KEY,
    user_id     TEXT NOT NULL,
    input       TEXT NOT NULL DEFAULT '{}',
    draft       TEXT NOT NULL,
    model       TEXT NOT NULL DEFAULT '',
    chapter_id  TEXT REFERENCES chapters(id) ON DELETE SET NULL,
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    applied_at  TEXT
);
CREATE INDEX IF NOT EXISTS idx_chapter_drafts_user ON chapter_drafts (user_id, created_at DESC);

-- concepts 재생성: origin 에 'chapter', 소유자·챕터 열 추가. NULL owner = 공용 시드(모두에게 보임).
CREATE TABLE concepts_v7 (
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
                        CHECK (origin IN ('seed', 'dynamic', 'chapter')),
    curriculum_version  TEXT,
    created_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at          TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    proposed_by         TEXT,
    owner_user_id       TEXT,
    chapter_id          TEXT REFERENCES chapters(id) ON DELETE SET NULL
);
INSERT INTO concepts_v7 (id, name, area, depth, summary, practical_cases, doc_links, status, origin,
                         curriculum_version, created_at, updated_at, proposed_by)
SELECT id, name, area, depth, summary, practical_cases, doc_links, status, origin,
       curriculum_version, created_at, updated_at, proposed_by
FROM concepts;
DROP TABLE concepts;
ALTER TABLE concepts_v7 RENAME TO concepts;
CREATE INDEX IF NOT EXISTS idx_concepts_area_status ON concepts (area, status);
CREATE INDEX IF NOT EXISTS idx_concepts_owner ON concepts (owner_user_id) WHERE owner_user_id IS NOT NULL;

-- 하루 학습량(분) — '다음 챕터 고르기' 제안 시점 계산(남은 분량 < 하루 학습량 × 14일).
ALTER TABLE user_learning_settings ADD COLUMN daily_minutes INTEGER NOT NULL DEFAULT 30
    CHECK (daily_minutes IN (15, 30, 45, 60));
