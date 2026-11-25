/** 강의 속 '직접 해 보기'·실습 시도(설계 v3 §2·§4). 숙달 근거가 아니라 기록·'고칠 곳' 신호.
 * pause·recall·summary 는 학습자가 한 번 눌러 스스로 채점한다(LLM 없음, 강의 품질 v2). */
export type ExerciseKind = "exercise" | "lab" | "pause" | "recall" | "summary";
export type ExerciseVerdict =
  | "correct" | "partial" | "incorrect" | "skipped"
  | "matched" | "different" | "unreadable"
  | "remembered" | "forgot" | "written" | "";
/** 예측 멈춤의 확신도 — 확인하기와 같은 값 */
export type PredictionConfidence = "sure" | "unsure" | "unknown";
export type ExerciseStatus = "pending" | "done" | "failed";

export interface ExerciseAttempt {
  id: string;
  lectureId: string;
  anchor: string;
  contentHash: string;
  kind: ExerciseKind;
  answer: string;
  verdict: ExerciseVerdict;
  feedback: string;
  status: ExerciseStatus;
  createdAt: string;
  confidence?: PredictionConfidence;
  /** 회상이 하루 뒤 복습에 넣은 문항 */
  reviewItemId?: string;
}

/** '고칠 곳' — 섹션(kind)과 이유 */
export interface SectionFix {
  section: string;
  reasons: string[];
}

export interface LectureExercises {
  attempts: ExerciseAttempt[];
  fixes: SectionFix[];
}

export interface ExerciseInput {
  anchor: string;
  contentHash: string;
  answer?: string;
  skip?: boolean;
  /** 스스로 채점(pause: correct|incorrect, recall: remembered|partial|forgot) */
  verdict?: ExerciseVerdict;
  confidence?: PredictionConfidence;
}

/** 한 블록의 읽기 신호(서버 store.ReadingSignal) — dwellMs 는 그 블록이 화면 중심이었던 시간, terms 는 연 용어별 횟수 */
export interface ReadingSignalInput {
  anchor: string;
  dwellMs: number;
  terms?: Record<string, number>;
}

export const EXERCISE_MAX_ANSWER_CHARS = 2000;
/** 예측 한 줄(서버 maxPredictionRunes) */
export const PREDICTION_MAX_CHARS = 300;
export const LAB_MAX_OUTPUT_BYTES = 8 * 1024;
