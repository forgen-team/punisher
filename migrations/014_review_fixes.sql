-- 숙달 리뷰 수정 (SQLite 포팅, 원본 014 — 설계 docs/plans/2026-11-punisher-mastery-design.md §4.2(이의) · §3.2(조회)).
-- ADD COLUMN 의 IF NOT EXISTS 만 뺀다(마이그레이션은 한 번만 실행). UUID 는 TEXT.

-- 이의 충돌 표지: 마지막으로 근거를 반영한 판정 턴. 근거 반영과 이의 복원만 바꾸고,
-- 통합 크레딧·대기 해제·stale 표시·다시 배우기는 건드리지 않는다(updated_at 비교의 오판 방지).
-- 비교값일 뿐이라 외래 키를 두지 않는다(턴이 지워져도 행은 그대로).
ALTER TABLE user_objective_mastery ADD COLUMN last_evidence_turn_id TEXT;

-- 사용자가 본 문항(복습 문항 고르기·변형 대상·'이미 앎' 확인 고르기·재사용 표시): user_id + item_id,
-- created_at 은 MAX(created_at)(마지막으로 본 시각)을 인덱스만으로 읽게 한다.
CREATE INDEX IF NOT EXISTS check_turns_user_item_idx ON check_turns (user_id, item_id, created_at);
-- 기억 유지 지표 표본(최근 n일의 판정 턴).
CREATE INDEX IF NOT EXISTS check_turns_user_created_idx ON check_turns (user_id, created_at);
