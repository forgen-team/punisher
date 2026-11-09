-- 강의 → "이 개념 더 깊게"(또는 '새 프롬프트로 다시 만들기')로 이어 본 강의 연결 (SQLite 포팅, 원본 004 + 005 의 kind 확장).
-- 강의는 불변·공유 레코드라 lectures 에 parent 열을 두지 않고 사용자 범위의 연결 테이블로 둔다.
-- 생성 중인 즉석 강의는 잡(child_job_id)으로 먼저 잇고, 강의가 저장되면 child_lecture_id 를 채운다.
CREATE TABLE IF NOT EXISTS lecture_links (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id            TEXT NOT NULL,
    parent_lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    child_lecture_id   TEXT REFERENCES lectures(id) ON DELETE CASCADE,
    child_job_id       TEXT REFERENCES generation_jobs(id) ON DELETE CASCADE,
    child_concept_id   TEXT NOT NULL,
    -- deeper: '이 개념 더 깊게' / regenerated: '새 프롬프트로 다시 만들기' (원본은 005 에서 추가)
    kind               TEXT NOT NULL DEFAULT 'deeper' CHECK (kind IN ('deeper', 'regenerated')),
    anchor             TEXT NOT NULL DEFAULT '',
    -- live: 사용자가 누를 때 기록 / inferred: 기존 데이터에서 추론(아래 백필)
    origin             TEXT NOT NULL DEFAULT 'live' CHECK (origin IN ('live', 'inferred')),
    created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    CONSTRAINT lecture_links_child CHECK (child_lecture_id IS NOT NULL OR child_job_id IS NOT NULL),
    CONSTRAINT lecture_links_not_self CHECK (child_lecture_id IS NULL OR child_lecture_id <> parent_lecture_id)
);
CREATE INDEX IF NOT EXISTS idx_lecture_links_parent ON lecture_links (user_id, parent_lecture_id, created_at);
CREATE INDEX IF NOT EXISTS idx_lecture_links_child ON lecture_links (user_id, child_lecture_id);
CREATE INDEX IF NOT EXISTS idx_lecture_links_job ON lecture_links (child_job_id) WHERE child_lecture_id IS NULL;

-- 기존(v0.1.0) 데이터 추론: 사용자가 요청해 만든 즉석 강의 L2 마다, 그 직전에 읽던 강의 L1 중
-- 본문에 L2 개념의 용어 블록(type=term, conceptId=L2.concept_id)이 있는 것을 원 강의로 본다.
-- "더 깊게"는 누르기 직전에 진도를 저장하므로 L1 의 마지막 방문이 L2 생성 30분 전 이후여야 한다.
INSERT INTO lecture_links (user_id, parent_lecture_id, child_lecture_id, child_concept_id, origin, created_at)
SELECT requested_by, parent_id, child_id, concept_id, 'inferred', created_at FROM (
    SELECT l2.requested_by, p1.lecture_id AS parent_id, l2.id AS child_id, l2.concept_id, l2.created_at,
           ROW_NUMBER() OVER (PARTITION BY l2.id ORDER BY p1.started_at DESC) AS rn
    FROM lectures l2
    JOIN user_lecture_progress p1 ON p1.user_id = l2.requested_by
    JOIN lectures l1 ON l1.id = p1.lecture_id
    WHERE l2.kind = 'instant' AND l2.requested_by IS NOT NULL
      AND l1.id <> l2.id AND l1.concept_id <> l2.concept_id
      AND p1.started_at <= l2.created_at
      AND p1.last_seen_at >= strftime('%Y-%m-%dT%H:%M:%fZ', l2.created_at, '-30 minutes')
      AND EXISTS (
          SELECT 1 FROM json_each(l1.content, '$.sections') sec, json_each(sec.value, '$.blocks') blk
          WHERE json_extract(blk.value, '$.type') = 'term' AND json_extract(blk.value, '$.conceptId') = l2.concept_id)
      AND NOT EXISTS (SELECT 1 FROM lecture_links k WHERE k.user_id = l2.requested_by AND k.child_lecture_id = l2.id)
) WHERE rn = 1;
