-- 생성 잡의 시도별 기록(시도 번호·결과 단계·오류 종류·메시지) (SQLite 포팅, 원본 008). 첫 시도가 마크업 해석에 실패하고
-- 두 번째에 성공해도 실패 시도가 남는다(라이브러리 검수 루프, 즉석 자동 재시도). JSONB → JSON TEXT.
ALTER TABLE generation_jobs ADD COLUMN attempt_log TEXT NOT NULL DEFAULT '[]';
