-- 강의 품질 v2 (조사 2026-11-25: 결정적 루브릭 G07~G23·감수 점수 E01~E13·읽는 중 회상·예측 멈춤 입력).
-- ADD COLUMN 만, 기존 행 변환 없음: 기본값이 지금 동작과 같다(기존 강의는 그대로 읽힌다).

-- 감수 기록의 품질 점수 JSON: {version, promptVersion, machine{rules,sql,arithmetic}, llm[{id,score,quote}], independent[], links[]}.
-- '{}' = 기록 전(이전 감수). 게이트에는 쓰지 않는다(추세·골든셋 비교용).
ALTER TABLE lecture_reviews ADD COLUMN quality TEXT NOT NULL DEFAULT '{}';

-- 강의 속 자기 채점 시도(kind pause | recall | summary)의 확신도(sure | unsure | unknown, '' = 묻지 않음).
ALTER TABLE lecture_exercise_attempts ADD COLUMN confidence TEXT NOT NULL DEFAULT '';
-- 회상(recall)이 하루 뒤 복습 큐에 넣은 문항(check_items.id). 숙달 근거가 아니다 — 복습에서 채점될 때만 반영된다.
ALTER TABLE lecture_exercise_attempts ADD COLUMN review_item_id TEXT;
