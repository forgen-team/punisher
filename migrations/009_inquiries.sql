-- 궁금한 것 질문 → 즉석 해설 (SQLite 포팅, 원본 009). 질문 하나에 해설 하나(캐시): 다시 열면 저장된 해설을 그대로 보이고
-- '다시 만들기'에서만 새로 만든다. 해설은 강의와 같은 섹션·블록 구조(content, JSON TEXT). concept_ids·proposed_concept_ids 는 JSON 배열.
-- read_at: '다 봤어요'(학습 홈 목록에서 빠짐 — 지도·기록에는 남는다).
CREATE TABLE IF NOT EXISTS inquiries (
    id                   TEXT PRIMARY KEY,
    user_id              TEXT NOT NULL,
    question             TEXT NOT NULL,
    status               TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'failed')),
    content              TEXT NOT NULL DEFAULT '{}',
    concept_ids          TEXT NOT NULL DEFAULT '[]',
    proposed_concept_ids TEXT NOT NULL DEFAULT '[]',
    model                TEXT NOT NULL DEFAULT '',
    prompt_version       TEXT NOT NULL DEFAULT '',
    error                TEXT NOT NULL DEFAULT '',
    attempt_log          TEXT NOT NULL DEFAULT '[]',
    read_at              TEXT,
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    answered_at          TEXT
);
CREATE INDEX IF NOT EXISTS idx_inquiries_user ON inquiries (user_id, created_at DESC);
