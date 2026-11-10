-- 이어서 묻기 (SQLite 포팅, 원본 010).
-- 이어서 묻기: 한 질문이 하나의 스레드. 후속 답도 강의 블록 구조(section, 섹션 하나)로 저장한다.
CREATE TABLE IF NOT EXISTS inquiry_followups (
    id                   TEXT PRIMARY KEY,
    inquiry_id           TEXT NOT NULL REFERENCES inquiries(id) ON DELETE CASCADE,
    user_id              TEXT NOT NULL,
    question             TEXT NOT NULL,
    quote                TEXT NOT NULL DEFAULT '',
    status               TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'answered', 'failed')),
    section              TEXT NOT NULL DEFAULT '{}',
    model                TEXT NOT NULL DEFAULT '',
    error                TEXT NOT NULL DEFAULT '',
    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    answered_at          TEXT
);
CREATE INDEX IF NOT EXISTS idx_inquiry_followups_thread ON inquiry_followups (inquiry_id, created_at);
-- 스레드당 답을 쓰는 중인 후속 질문은 하나(동시 요청도 DB 에서 막는다).
CREATE UNIQUE INDEX IF NOT EXISTS idx_inquiry_followups_one_pending ON inquiry_followups (inquiry_id) WHERE status = 'pending';
