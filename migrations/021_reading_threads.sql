-- 읽기 스레드(설계 docs/plans/2026-11-learner-level-design.md ④): 기존 lecture_questions 를 넓혀
-- 구절 스레드로 쓴다. ADD COLUMN 만, 표 재생성 없음(kind CHECK 은 그대로 — 스레드 질문은 kind='question'),
-- 기존 행 변환 없음(기존 질문은 quote '' · 루트 · mode 'question').

-- 고른 구절(루트에만; 이어 묻기는 루트의 구절을 쓴다).
ALTER TABLE lecture_questions ADD COLUMN quote TEXT NOT NULL DEFAULT '';
-- 이어 묻기면 루트 질문 id(NULL = 루트). 이어 묻기의 이어 묻기도 늘 루트를 가리킨다.
ALTER TABLE lecture_questions ADD COLUMN parent_id TEXT;
-- easier | deeper | question (앱이 검증한다).
ALTER TABLE lecture_questions ADD COLUMN mode TEXT NOT NULL DEFAULT 'question';

CREATE INDEX IF NOT EXISTS idx_lecture_questions_parent ON lecture_questions (parent_id, created_at)
    WHERE parent_id IS NOT NULL;
-- 수준 보정 신호(최근 7일 '더 쉽게' 횟수)를 싸게 센다.
CREATE INDEX IF NOT EXISTS idx_lecture_questions_user_easier ON lecture_questions (user_id, created_at)
    WHERE mode = 'easier';

-- 수준 보정 제안을 닫은 때(7일간 숨김). NULL = 닫은 적 없음.
ALTER TABLE user_learning_settings ADD COLUMN level_suggestion_dismissed_at TEXT;
-- 설명 수준을 바꾼 때: 제안 신호는 그 뒤의 '더 쉽게'만 센다(바꾸자마자 다시 낮추라고 하지 않게).
ALTER TABLE user_learning_settings ADD COLUMN reader_level_changed_at TEXT;
