-- 사용자별 강의 프롬프트 지시문 블록 오버라이드(설정 > 강의 프롬프트) (SQLite 포팅, 원본 005).
-- 행이 없으면 코드 기본값(prompts.Blocks[].Default). 단독판은 사용자가 한 명이지만 원본과 같은 모양을 유지한다.
CREATE TABLE IF NOT EXISTS prompt_overrides (
    user_id     TEXT NOT NULL,
    prompt_id   TEXT NOT NULL,
    body        TEXT NOT NULL,
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, prompt_id)
);

-- 저장·되돌리기 이력. (사용자, 블록)별 최근 20개만 남긴다(store 에서 정리).
CREATE TABLE IF NOT EXISTS prompt_override_history (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     TEXT NOT NULL,
    prompt_id   TEXT NOT NULL,
    body        TEXT NOT NULL,
    action      TEXT NOT NULL CHECK (action IN ('save', 'reset', 'restore')),
    created_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS idx_prompt_override_history ON prompt_override_history (user_id, prompt_id, id DESC);

-- 용어 설명 프롬프트를 바꾼 사용자 전용 캐시(공용 term_glosses 는 기본 프롬프트 결과만).
CREATE TABLE IF NOT EXISTS user_term_glosses (
    user_id      TEXT NOT NULL,
    concept_id   TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    prompt_hash  TEXT NOT NULL,
    gloss        TEXT NOT NULL,
    model        TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, concept_id, prompt_hash)
);
