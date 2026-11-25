-- 블록 단위 읽기 신호(조사 #15): 블록 체류 시간과 용어 팝오버를 연 횟수. 이 기기 사용자 한 명의 기록이며
-- 강의 '고칠 곳'·다시 만들기 신호와 품질 하네스(감수 '막힌 곳' 대조)에만 쓴다. 기존 행 변환 없음.
CREATE TABLE IF NOT EXISTS lecture_reading_signals (
    user_id     TEXT NOT NULL,
    lecture_id  TEXT NOT NULL REFERENCES lectures(id) ON DELETE CASCADE,
    anchor      TEXT NOT NULL,
    dwell_ms    INTEGER NOT NULL DEFAULT 0,
    term_opens  INTEGER NOT NULL DEFAULT 0,
    terms       TEXT NOT NULL DEFAULT '{}',   -- {개념 id: 연 횟수}
    updated_at  TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now')),
    PRIMARY KEY (user_id, lecture_id, anchor)
);
