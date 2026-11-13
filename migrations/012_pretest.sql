-- 사전 진단(P3) (SQLite 포팅, 원본 012 — 설계 docs/plans/2026-11-punisher-mastery-design.md §3.1 · §5 P3).
-- 새 kind(pretest·known_verify·placement)·purpose·format(choice)·outcome(recorded)은 CHECK 가 없어 열 추가만 한다(§0.3).
-- BOOLEAN → INTEGER(0/1), ADD COLUMN IF NOT EXISTS → ADD COLUMN(마이그레이션은 한 번만 실행).

-- 강의 전 예측 질문 켜기/끄기(설정 > 학습).
ALTER TABLE user_learning_settings ADD COLUMN pretest_enabled INTEGER NOT NULL DEFAULT 1;

-- 진단 대기열: 일괄 '이미 앎'으로 표시한 목표 — 복습 세션에 하루 1~2개씩 known_verify 문항으로 섞는다.
ALTER TABLE user_objective_mastery ADD COLUMN verify_queued INTEGER NOT NULL DEFAULT 0;
