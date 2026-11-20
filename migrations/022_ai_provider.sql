-- AI 선택(설계 docs/plans/2026-11-learner-level-design.md ⑥): 강의 생성·검수·질문 답·채점에 쓸 AI 하나.
-- ADD COLUMN 만, 기존 행 변환 없음(지금까지는 모두 Claude Code 였다).

-- claude | codex | agy (앱이 검증한다; 모르는 값은 claude 로 읽는다).
ALTER TABLE user_learning_settings ADD COLUMN ai_provider TEXT NOT NULL DEFAULT 'claude';
-- 만든 AI(기록용). 재사용 조건에는 넣지 않는다 — 같은 수준·언어면 어떤 AI 가 썼든 재사용.
ALTER TABLE lectures ADD COLUMN provider TEXT NOT NULL DEFAULT 'claude';
ALTER TABLE generation_jobs ADD COLUMN provider TEXT NOT NULL DEFAULT 'claude';
