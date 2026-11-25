-- 학습 기능(W5a): 반복 실패·보류·FSRS 기억 상태, 학습 선호, 목표 기한, 복습 카드.
-- 새 표만 만들고 기존 표·행은 바꾸지 않는다. 기존 목표는 FSRS 기록이 없어(reps 0) 상자 간격 그대로 시작하고,
-- 판정이 쌓일수록 FSRS 간격과 섞인다(learning.BlendDays).

-- 목표별 기억 상태 — 상자·등급(user_objective_mastery)과 따로 둔다(이의 제기 복원 스냅숏은 상자만 되돌린다).
CREATE TABLE IF NOT EXISTS user_objective_memory (
    user_id        TEXT NOT NULL,
    objective_id   TEXT NOT NULL REFERENCES objectives(id) ON DELETE CASCADE,
    stability      REAL NOT NULL DEFAULT 0,   -- FSRS S(일). 0 = 아직 없음
    difficulty     REAL NOT NULL DEFAULT 0,   -- FSRS D(1~10). 0 = 아직 없음
    reps           INTEGER NOT NULL DEFAULT 0,
    last_review_at TEXT,
    fail_count     INTEGER NOT NULL DEFAULT 0, -- 누적 실패(도움 받은 정답 포함). 맞혀도 줄지 않는다
    leech_at       TEXT,                       -- 반복 실패로 표시한 때(처리하면 NULL)
    suspended_at   TEXT,                       -- 잠시 보류: 복습에서 뺀다(되돌리기 가능)
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, objective_id)
);
CREATE INDEX IF NOT EXISTS user_objective_memory_flags ON user_objective_memory (user_id)
    WHERE leech_at IS NOT NULL OR suspended_at IS NOT NULL;

-- 학습 선호(설정 > 학습). user_learning_settings 와 따로 둔다(기존 upsert 를 넓히지 않는다).
CREATE TABLE IF NOT EXISTS user_study_prefs (
    user_id           TEXT PRIMARY KEY,
    memory_model      TEXT NOT NULL DEFAULT 'fsrs' CHECK (memory_model IN ('ladder', 'fsrs')),
    target_retention  REAL NOT NULL DEFAULT 0.9,
    rest_days         TEXT NOT NULL DEFAULT '[]',  -- 쉬는 요일 0=일 … 6=토
    reminder_hour     INTEGER NOT NULL DEFAULT -1, -- 복습 알림 시각(KST 0~23), -1 = 끔
    last_reminded_day TEXT NOT NULL DEFAULT '',
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- 지도 목표의 기한(YYYY-MM-DD, KST).
CREATE TABLE IF NOT EXISTS goal_deadlines (
    goal_id    TEXT PRIMARY KEY REFERENCES user_goals(id) ON DELETE CASCADE,
    user_id    TEXT NOT NULL,
    due_on     TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

-- 메모·구절 대화에서 사용자가 고른 것만 만든 복습 카드(질문·답은 사용자가 확인·고친 글).
CREATE TABLE IF NOT EXISTS review_cards (
    id             TEXT PRIMARY KEY,
    user_id        TEXT NOT NULL,
    source_kind    TEXT NOT NULL CHECK (source_kind IN ('note', 'thread')),
    source_id      TEXT NOT NULL,
    lecture_id     TEXT REFERENCES lectures(id) ON DELETE SET NULL,
    concept_id     TEXT REFERENCES concepts(id) ON DELETE SET NULL,
    anchor         TEXT NOT NULL DEFAULT '',
    question       TEXT NOT NULL,
    answer         TEXT NOT NULL,
    stability      REAL NOT NULL DEFAULT 0,
    difficulty     REAL NOT NULL DEFAULT 0,
    reps           INTEGER NOT NULL DEFAULT 0,
    lapses         INTEGER NOT NULL DEFAULT 0,
    next_review_at TEXT,
    last_review_at TEXT,
    suspended_at   TEXT,
    created_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    updated_at     TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    UNIQUE (user_id, source_kind, source_id)
);
CREATE INDEX IF NOT EXISTS review_cards_due ON review_cards (user_id, next_review_at);

CREATE TABLE IF NOT EXISTS review_card_log (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    card_id     TEXT NOT NULL REFERENCES review_cards(id) ON DELETE CASCADE,
    user_id     TEXT NOT NULL,
    rating      INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 3),
    reviewed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
CREATE INDEX IF NOT EXISTS review_card_log_user_day ON review_card_log (user_id, reviewed_at);
-- 오늘 목표·주간 활동이 확인 문항 끝(outcome)을 날짜별로 센다.
CREATE INDEX IF NOT EXISTS check_turns_user_outcome_day ON check_turns (user_id, created_at) WHERE outcome IS NOT NULL;
