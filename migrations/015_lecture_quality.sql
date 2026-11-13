-- 강의 품질 고도화 (SQLite 포팅, 원본 015 — 설계 docs/plans/2026-11-punisher-lecture-quality-design.md §2.2(소급 검수) · §5(시간 계산)).
-- ADD COLUMN 의 IF NOT EXISTS 만 뺀다(마이그레이션은 한 번만 실행). BOOLEAN 은 INTEGER 0/1, TIMESTAMPTZ 는 UTC TEXT.

-- estimated_minutes 가 learning.EstimateMinutes(본문 분량 계산)로 기록됐는지. 0(기존 강의)은 섹션 분 합계라
-- 조회 때 본문으로 다시 계산한다(값을 고쳐 쓰지 않는다).
ALTER TABLE lectures ADD COLUMN minutes_estimated INTEGER NOT NULL DEFAULT 0;

-- 소급 검수 대기열 앞쪽: 검수 통과 기록이 없는 강의에서 '확인하기'를 누른 시각(409 LECTURE_NOT_REVIEWED).
ALTER TABLE lectures ADD COLUMN review_requested_at TEXT;

-- 검수 통과 기록 조회(강의 응답의 reviewState·확인 문항 생성 조건·소급 검수 대상).
CREATE INDEX IF NOT EXISTS lecture_reviews_pass_idx ON lecture_reviews (lecture_id) WHERE verdict = 'pass';
