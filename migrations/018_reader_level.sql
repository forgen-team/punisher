-- Punisher 학습자 수준(설계 docs/plans/2026-11-learner-level-design.md ①).
-- 017 과 같은 방식: ADD COLUMN 만, 표 재생성 없음, 기존 행 변환 없음.
-- 수준은 설명 밀도만 바꾼다(가르치는 사실·목표는 같다). 기존 강의·잡은 기본값 'practitioner' = 지금까지의 독자.

-- 사용자 설명 수준(beginner|novice|practitioner|advanced). '' = 아직 정하지 않음(웹이 처음 설정 시트를 띄운다).
ALTER TABLE user_learning_settings ADD COLUMN reader_level TEXT NOT NULL DEFAULT '';
-- 처음 설정의 영역별 아는 정도: {"<area>": "new|heard|can_explain"} JSON TEXT.
ALTER TABLE user_learning_settings ADD COLUMN area_familiarity TEXT NOT NULL DEFAULT '{}';

-- 강의 수준: 같은 대상이라도 수준이 다르면 다른 강의. 버전 번호는 017 의 언어처럼 수준을 가로질러 이어진다.
ALTER TABLE lectures ADD COLUMN reader_level TEXT NOT NULL DEFAULT 'practitioner';
-- 대상 조회 인덱스는 이름을 그대로 두고 수준을 더해 다시 만든다(활성 잡 인덱스와 같은 방식).
DROP INDEX IF EXISTS lectures_target_language_idx;
CREATE INDEX IF NOT EXISTS lectures_target_language_idx
    ON lectures (concept_id, objective_key, depth, language, reader_level, status);

-- 생성 잡 수준: 같은 대상·언어라도 수준이 다르면 동시에 돌 수 있다(부분 유니크 인덱스 교체).
ALTER TABLE generation_jobs ADD COLUMN reader_level TEXT NOT NULL DEFAULT 'practitioner';
DROP INDEX IF EXISTS generation_jobs_one_active_target;
CREATE UNIQUE INDEX IF NOT EXISTS generation_jobs_one_active_target
    ON generation_jobs (objective_key, depth, language, reader_level)
    WHERE status IN ('pending', 'running') AND kind IN ('library', 'instant');
