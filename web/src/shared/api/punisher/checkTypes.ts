/** 숙달 확인(P1)·사전 진단(P3) — punisher-server models/checks.go 와 같은 형태. */

export type CheckSessionStatus = "preparing" | "active" | "finished" | "abandoned" | "failed";
export type CheckItemFormat = "recall" | "predict" | "apply" | "integrative" | "choice";
/** 세션 종류 — 확인 · 복습 · 사전 진단(예측 질문 · 이미 앎 확인 · 배치 진단) */
export type CheckKind = "check" | "review" | "pretest" | "known_verify" | "placement" | "synthesis";
/** 답 전에 고르는 확신도(v3 §3) — 판정은 바꾸지 않는다. */
export type Confidence = "sure" | "unsure" | "unknown";
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
  | "dispute"
  | "dont_know";
export type CheckTurnStatus = "pending" | "ready" | "failed";
export type CheckVerdict = "correct" | "partial" | "incorrect" | "revealed" | "held";
/** recorded: 예측·배치 답(기록만, 숙달 근거 아님) */
export type CheckOutcome = "unaided_correct" | "unaided_partial" | "aided" | "failed" | "held" | "recorded";
/** none: 기록만 하는 세션(예측 질문 · 배치 진단) */
export type CheckGrader = "llm" | "self" | "llm_regrade" | "none";
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
  /** '이미 앎' 확인에서 통과하지 못함 → '다시 배우기' 제안 */
  relearn?: boolean;
  /** 확실했는데 틀림('착각', v3 §3) */
  confidentError?: boolean;
}

export interface CheckSession {
  id: string;
  kind: CheckKind;
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
  /** choice 문항의 보기(3~4개) */
  choices?: string[];
  /** 장면의 로그·숫자(종합 문제 등) */
  log?: string;
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
  /** 그 강의의 '한 장 요약'(마크다운) — 확인·복습의 끝난 문항에만, 요약 없는 강의는 생략 */
  lectureSummary?: string;
}

export interface CheckGrade {
  met: string[] | null;
  missing: string[] | null;
  quote: string;
  misconception: string;
  confidence: number;
  /** choice 문항에서 고른 보기(0부터) */
  choice?: number;
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
  /** 학습자 답(첫 답)의 확신도 */
  confidence?: Confidence;
  /** 확실했는데 틀림('착각') — 문항의 마지막 판정 턴 */
  confidentError?: boolean;
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
  /** 준비 중인 확인 세션의 문항 생성 작업 id([취소] 대상). 그 밖에는 없음 */
  generationJobId?: string;
  mastery: ObjectiveMastery[] | null;
  /** 확인 세션: 같은 목표의 처음 예측(예측 질문 답) */
  predictions?: Prediction[];
  /** 배치 진단 세션: 챕터와 이미 앎 후보 */
  placement?: PlacementResult;
  /** 이 세션에서 오개념을 인정한 턴 */
  misconceptionTurnIds?: string[] | null;
}

/** 강의 전 예측 질문에 남긴 답(목표별 처음 예측). */
export interface Prediction {
  objectiveId: string;
  prompt: string;
  answer: string;
  dontKnow: boolean;
  at: string;
}

export interface PlacementCandidate {
  conceptId: string;
  conceptName: string;
}

/** 배치 진단 결과 — 맞힌 개념 = 이미 앎 후보(확정해야 known). */
export interface PlacementResult {
  chapterId: string;
  candidates: PlacementCandidate[];
}

/** POST /checks 의 사전 진단 요청(pretest 는 lectureId 또는 jobId). */
export type DiagnosticRequest =
  | { kind: "pretest"; lectureId: string }
  | { kind: "pretest"; jobId: string }
  | { kind: "known_verify"; conceptId: string }
  | { kind: "placement"; chapterId: string }
  | { kind: "synthesis" };

/** 사전 진단 문항의 답 — 보기 번호 또는 '모르겠어요'. */
export type DiagnosticChoice = { choice: number } | { dontKnow: true };

/** 답·이의 사유 길이 상한(서버 services.MaxCheckAnswerRunes · MaxDisputeRunes). */
export const CHECK_MAX_ANSWER_CHARS = 2000;
export const CHECK_MAX_DISPUTE_CHARS = 500;
/** 끝난 세션의 이의 제기 가능 시간(서버 disputeWindow). */
export const CHECK_DISPUTE_WINDOW_MS = 24 * 60 * 60 * 1000;
