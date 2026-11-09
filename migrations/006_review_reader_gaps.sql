-- 감수의 '초보 독자 시뮬레이션' 결과(정의 없이 전제한 문장 목록)를 감수 기록과 함께 남긴다 (SQLite 포팅, 원본 006).
-- 강의 품질 감사(설명 없는 전제 추적)에 쓴다.
ALTER TABLE lecture_reviews ADD COLUMN reader_gaps TEXT NOT NULL DEFAULT '[]';
