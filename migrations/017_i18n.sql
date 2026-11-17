-- Punisher 한국어·영어 (SQLite 포팅, 원본 017 — 설계 docs/plans/2026-11-punisher-i18n-design.md §2·§9).
-- ADD COLUMN 의 IF NOT EXISTS 만 뺀다(러너가 한 번만 실행). JSONB 는 JSON TEXT. 표 재생성 없음.
-- 기존 행 변환 없음: 모든 언어 열의 기본값 'ko' = 지금까지 만든 것.
-- 새 사용자의 기본 locale 은 열 기본값이 아니라 store 의 기본 설정(OS 로캘, config)이 정한다.

-- 화면·생성 언어(ko|en). example_language(예시 코드 언어)와 다르다.
ALTER TABLE user_learning_settings ADD COLUMN locale TEXT NOT NULL DEFAULT 'ko';

-- 강의 언어. 버전 유일 제약은 그대로: 버전 번호는 언어를 가로질러 이어진다(§2).
ALTER TABLE lectures ADD COLUMN language TEXT NOT NULL DEFAULT 'ko';
CREATE INDEX IF NOT EXISTS lectures_target_language_idx
    ON lectures (concept_id, objective_key, depth, language, status);

-- 확인 문항 언어 + 영어 목표 문장 스냅숏(은퇴 비교, §9).
ALTER TABLE check_items ADD COLUMN language TEXT NOT NULL DEFAULT 'ko';
ALTER TABLE check_items ADD COLUMN objective_text_en TEXT NOT NULL DEFAULT '';

-- 생성 잡 언어: 같은 대상이라도 언어가 다르면 동시에 돌 수 있다(부분 유니크 인덱스 교체).
ALTER TABLE generation_jobs ADD COLUMN language TEXT NOT NULL DEFAULT 'ko';
DROP INDEX IF EXISTS generation_jobs_one_active_target;
CREATE UNIQUE INDEX IF NOT EXISTS generation_jobs_one_active_target
    ON generation_jobs (objective_key, depth, language)
    WHERE status IN ('pending', 'running') AND kind IN ('library', 'instant');

-- 커리큘럼 영어판(curriculum/i18n/en 오버레이를 시드가 채운다). 비면 한국어로 대체.
ALTER TABLE concepts ADD COLUMN name_en TEXT NOT NULL DEFAULT '';
ALTER TABLE concepts ADD COLUMN summary_en TEXT NOT NULL DEFAULT '';
ALTER TABLE concepts ADD COLUMN practical_cases_en TEXT NOT NULL DEFAULT '[]';
ALTER TABLE objectives ADD COLUMN text_en TEXT NOT NULL DEFAULT '';
