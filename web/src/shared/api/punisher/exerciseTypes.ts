/** 강의 속 '직접 해 보기'·실습 시도(설계 v3 §2·§4). 숙달 근거가 아니라 기록·'고칠 곳' 신호. */
export type ExerciseKind = "exercise" | "lab";
export type ExerciseVerdict =
  | "correct" | "partial" | "incorrect" | "skipped"
  | "matched" | "different" | "unreadable" | "";
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
}

export const EXERCISE_MAX_ANSWER_CHARS = 2000;
export const LAB_MAX_OUTPUT_BYTES = 8 * 1024;
