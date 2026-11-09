-- 학습 UX 개선: 이어보기 제거(소프트) · 사용자 학습 설정(예시 코드 언어) (SQLite 포팅, 원본 003)

-- 이어보기에서 제거: 강의·진도는 지우지 않고 표시만 한다(되돌리기 가능).
ALTER TABLE user_lecture_progress ADD COLUMN dismissed_at TEXT;
CREATE INDEX IF NOT EXISTS idx_user_lecture_progress_active
    ON user_lecture_progress (user_id, last_seen_at DESC)
    WHERE completed_at IS NULL AND dismissed_at IS NULL;

-- 사용자별 학습 설정. 행이 없으면 기본값(learning.DefaultExampleLanguage).
CREATE TABLE IF NOT EXISTS user_learning_settings (
    user_id           TEXT PRIMARY KEY,
    example_language  TEXT NOT NULL,
    updated_at        TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);
