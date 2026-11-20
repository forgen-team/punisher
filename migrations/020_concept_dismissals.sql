-- '필요 없음'(설계 docs/plans/2026-11-learner-level-design.md ③): 사용자가 개념을 직접 뺀다.
-- 이미 앎·완료(user_concept_states.state)와 별개인 표 — 상태·숙달·선행 잠금 계산은 그대로 두고,
-- 추천·다음 개념·미리 만들기·복습에서만 뺀다. 행이 있으면 뺀 것, 되돌리면 행을 지운다.
-- 새 표만 만든다(기존 표 재생성·행 변환 없음).

CREATE TABLE user_concept_dismissals (
    user_id       TEXT NOT NULL,
    concept_id    TEXT NOT NULL REFERENCES concepts(id) ON DELETE CASCADE,
    dismissed_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, concept_id)
);
