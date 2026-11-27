-- 카드 삭제 되돌리기: 지운 카드는 행을 지우지 않고 deleted_at 만 적는다(복습 기록 review_card_log 는 그대로).
-- 목록·복습 대상·개수는 deleted_at IS NULL 인 카드만 센다. 지운 지 30일이 지나면 행은 두고 개인 내용(앞·뒤 글·출처)만 비우고
-- purged_at 에 적는다 — 복습 기록이 가리키는 카드 행이 남아 외래 키·통계가 그대로다.
-- 열·색인 추가만 — 기존 행·열은 바꾸지 않는다(새 열은 NULL = 지우지 않음·비우지 않음).
ALTER TABLE review_cards ADD COLUMN deleted_at TEXT;
ALTER TABLE review_cards ADD COLUMN purged_at TEXT;
CREATE INDEX IF NOT EXISTS review_cards_deleted ON review_cards (user_id, deleted_at) WHERE deleted_at IS NOT NULL;
