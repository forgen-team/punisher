/** 숙달 확인(P1) — punisher-server models/checks.go 와 같은 형태. */

export type CheckSessionStatus = "preparing" | "active" | "finished" | "abandoned" | "failed";
export type CheckItemFormat = "recall" | "predict" | "apply" | "integrative";
export type CheckTurnRole = "tutor" | "learner";
export type CheckMove =
  | "ask"
  | "narrow"
  | "explain"
  | "reask"
  | "challenge"
  | "next"
  | "wrap"
  | "answer"
  | "reveal"
  | "self_grade"
  | "dispute";
export type CheckTurnStatus = "pending" | "ready" | "failed";
export type CheckVerdict = "correct" | "partial" | "incorrect" | "revealed" | "held";
export type CheckOutcome = "unaided_correct" | "unaided_partial" | "aided" | "failed" | "held";
export type CheckGrader = "llm" | "self" | "llm_regrade";
export type MasteryBest = "none" | "revealed" | "aided" | "unaided";
/** 0 미확인 · 1 들어봄 · 2 따라감 · 3 혼자 답함 · 4 기억함 · 5 설명 가능 */
export type MasteryLevel = 0 | 1 | 2 | 3 | 4 | 5;

/** 목표 하나의 숙달(GET /mastery 의 objectives, CheckDetail.mastery). */
export interface ObjectiveMastery {
  objectiveId: string;
  conceptId: string;
  text: string;
  depth: number;
  ordinal: number;
  level: number;
  box: number;
  gradStep: number;
  best: MasteryBest;
  integrativeOk: boolean;
  lapses: number;
  nextReviewAt: string | null;
  stale: boolean;
  /** 복습이 밀려 첫 복습일 미정(대기) */
  queued: boolean;
  /** 마지막 근거의 채점 주체("" = 없음) */
  lastGrader: CheckGrader | "";
  updatedAt: string | null;
}

/** GET /mastery?conceptId= · POST /objectives/{id}/relearn */
export interface ConceptMastery {
  conceptId: string;
  /** 활성 목표 레벨의 최솟값 */
  level: number;
  mastered: boolean;
  masteryTarget: number;
  /** 근거가 없는 활성 목표 수("새 목표 n개 미확인") */
  uncheckedCount: number;
  objectives: ObjectiveMastery[];
}

/** 결과 요약 한 줄(문항 하나 = 목표 하나). */
export interface CheckResultItem {
  itemId: string;
  /** "" = 통합 문항 */
  objectiveId: string;
  outcome: CheckOutcome;
  levelBefore: number;
  levelAfter: number;
  missing: string[] | null;
  anchor: string;
  misconception: string;
  nextReviewAt: string | null;
  lapses: number;
  /** 복습이 밀려 대기(서버가 아직 안 주면 숙달의 queued 로 판단) */
  queued?: boolean;
}

export interface CheckSession {
  id: string;
  kind: "check" | "review";
  grader: CheckGrader;
  conceptId: string;
  lectureId: string;
  itemIds: string[];
  /** 현재 문항 위치(0부터) */
  cursor: number;
  status: CheckSessionStatus;
  result: { items?: CheckResultItem[] } | null;
  error?: string;
  createdAt: string;
  updatedAt: string;
  finishedAt?: string | null;
}

export interface CheckMaterial {
  code?: string;
  lang?: string;
}

export interface CheckItem {
  id: string;
  /** "" = 통합 문항 */
  objectiveId: string;
  format: CheckItemFormat;
  prompt: string;
  material?: CheckMaterial | null;
  done: boolean;
  outcome?: CheckOutcome;
  /** answer·must·anchor 는 끝난 문항에만 */
  answer?: string;
  must?: string[];
  anchor?: string;
}

export interface CheckGrade {
  met: string[] | null;
  missing: string[] | null;
  quote: string;
  misconception: string;
  confidence: number;
  /** 이의 제기 턴에서만 */
  accepted?: boolean;
  conflict?: boolean;
}

export interface CheckTurn {
  id: string;
  itemId: string;
  seq: number;
  role: CheckTurnRole;
  move: CheckMove | "";
  text: string;
  status: CheckTurnStatus;
  verdict?: CheckVerdict;
  outcome?: CheckOutcome;
  aided: boolean;
  grader?: CheckGrader;
  reusedItem: boolean;
  grade?: CheckGrade;
  model?: string;
  error?: string;
  createdAt: string;
}

export interface CheckCurrent {
  itemId: string;
  answers: number;
  canAnswer: boolean;
  canReveal: boolean;
  canSelfGrade: boolean;
}

/** 모든 확인 동작의 응답(pending=true 면 202 — 끝날 때까지 GET 으로 다시 확인). */
export interface CheckDetail {
  session: CheckSession;
  items: CheckItem[];
  turns: CheckTurn[];
  /** 세션이 active 가 아니면 null */
  current: CheckCurrent | null;
  pending: boolean;
  mastery: ObjectiveMastery[] | null;
}

/** 답·이의 사유 길이 상한(서버 services.MaxCheckAnswerRunes · MaxDisputeRunes). */
export const CHECK_MAX_ANSWER_CHARS = 2000;
export const CHECK_MAX_DISPUTE_CHARS = 500;
/** 끝난 세션의 이의 제기 가능 시간(서버 disputeWindow). */
export const CHECK_DISPUTE_WINDOW_MS = 24 * 60 * 60 * 1000;
