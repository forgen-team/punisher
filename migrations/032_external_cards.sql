-- punisher:foreign_keys=off
-- 외부 카드(atlas 등 다른 앱이 보낸 복습 카드): review_cards.source_kind 에 'external' 을 허용하고
-- 출처 링크(source_url)·외부 id(external_id, 예 'atlas:<cardId>', 사용자별 유일)·근거 바뀜 표시(stale)를 더한다.
-- SQLite 는 CHECK 를 바꾸려면 표를 다시 만들어야 한다(019 와 같은 12단계 재생성).
-- 기존 행·열·인덱스는 그대로 옮기고(새 열은 NULL/0), review_card_log 의 참조는 이름으로 이어진다.

CREATE TABLE review_cards_v32 (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    source_kind    TEXT NOT NULL CHECK (source_kind IN ('note', 'thread', 'external')),
    source_id      TEXT NOT NULL,
    lecture_id     TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    concept_id     TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    anchor         TEXT NOT NULL DEFAULT '',
    question       TEXT NOT NULL,
    answer         TEXT NOT NULL,
    stability      REAL NOT NULL DEFAULT 0,
    difficulty     REAL NOT NULL DEFAULT 0,
    reps           INTEGER NOT NULL DEFAULT 0,
    lapses         INTEGER NOT NULL DEFAULT 0,
    next_review_at TEXT,
    last_review_at TEXT,
    suspended_at   TEXT,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    source_url     TEXT,
    external_id    TEXT,
    stale          INTEGER NOT NULL DEFAULT 0,
    UNIQUE (user_id, source_kind, source_id)
);
INSERT INTO review_cards_v32 (id, user_id, source_kind, source_id, lecture_id, concept_id, anchor, question, answer,
                              stability, difficulty, reps, lapses, next_review_at, last_review_at, suspended_at, created_at, updated_at)
SELECT id, user_id, source_kind, source_id, lecture_id, concept_id, anchor, question, answer,
       stability, difficulty, reps, lapses, next_review_at, last_review_at, suspended_at, created_at, updated_at
FROM review_cards;
DROP TABLE review_cards;
ALTER TABLE review_cards_v32 RENAME TO review_cards;
CREATE INDEX IF NOT EXISTS review_cards_due ON review_cards (user_id, next_review_at);
CREATE UNIQUE INDEX IF NOT EXISTS review_cards_external ON review_cards (user_id, external_id) WHERE external_id IS NOT NULL;
