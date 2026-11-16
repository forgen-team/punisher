import type { BadgeTone } from "@/src/shared/ui/kit";
import type { CheckOutcome, CheckSessionStatus, CheckVerdict, Confidence } from "@/src/shared/api/punisher";

/** 확신도(v3 §3) */
export const CONFIDENCE_ORDER: readonly Confidence[] = ["sure", "unsure", "unknown"];
export const CONFIDENCE_LABELS: Record<Confidence, string> = { sure: "확실", unsure: "대충", unknown: "모름" };

export const VERDICT_TONE: Record<CheckVerdict, BadgeTone> = {
  correct: "success",
  partial: "warning",
  incorrect: "danger",
  revealed: "neutral",
  held: "neutral",
};

export const OUTCOME_TONE: Record<CheckOutcome, BadgeTone> = {
  "unaided_correct": "success",
  "unaided_partial": "warning",
  aided: "warning",
  failed: "danger",
  held: "neutral",
  recorded: "neutral",
};

/** 이의 제기 재채점 결과 */
export const DISPUTE_RESULT_LABELS: Record<"accepted" | "rejected" | "conflict", string> = {
  accepted: "이의 받아들임",
  rejected: "원래 판정 유지",
  /** 받아들였지만 그 뒤 다른 근거가 있어 상자는 되돌리지 않음 */
  conflict: "이후 기록이 있어 상자 유지",
};

/** 실행 중 표시(튜터 턴 · 이의 재채점) */
export const PENDING_LABELS: Record<"turn" | "dispute", string> = {
  turn: "튜터가 보는 중이에요",
  dispute: "다시 채점하는 중이에요",
};

/** 세션 상태 줄(active 는 표시 없음) */
export const SESSION_STATUS_LABELS: Record<CheckSessionStatus, string> = {
  preparing: "문항을 만드는 중이에요",
  active: "",
  finished: "확인을 마쳤어요",
  abandoned: "포기한 확인이에요",
  failed: "문항을 만들지 못했어요",
};

/** 복습 세션 상태 줄(active 는 표시 없음) */
export const REVIEW_STATUS_LABELS: Record<CheckSessionStatus, string> = {
  preparing: "복습 문항을 준비하는 중이에요",
  active: "",
  finished: "복습을 마쳤어요",
  abandoned: "그만둔 복습이에요",
  failed: "복습 문항을 준비하지 못했어요",
};

/** 결과의 "처음 예측 → 지금 답" */
export const PREDICTION_LABELS = {
  before: "처음 예측",
  now: "지금 답",
} as const;

/** 상단 제목(확인·복습 외 진단 세션) */
export const DIAGNOSTIC_TITLES = {
  pretest: "예측 질문",
  "known_verify": "이미 앎 확인",
  placement: "배치 진단",
  synthesis: "이번 주 종합 문제",
} as const;

/** 종합 문제 결과(v3 §5) — 상자는 움직이지 않는다. */
export const SYNTHESIS_RESULT_LABELS = {
  passed: "통과 — 함께 설명할 수 있어요",
  pulled: "관련 복습을 앞당겼어요",
  recorded: "기록만 남겼어요",
} as const;

export const CONFIDENT_ERROR_TITLE = "착각";

/** 배치 진단 결과 */
export const PLACEMENT_LABELS = {
  candidates: "이미 앎 후보",
  none: "이미 앎 후보가 없어요",
  confirm: "이미 앎으로 표시",
  confirmed: "이미 앎으로 표시함",
  map: "지도에서 보기",
  failed: (n: number) => `${n}개를 표시하지 못했어요. 다시 눌러 주세요.`,
} as const;

/** 결과의 드러난 오개념 · 인정 버튼 */
export const MISCONCEPTION_LABELS = {
  revealed: "드러난 오개념",
  acknowledge: "맞아요, 그렇게 알았어요",
  acknowledged: "인정함",
} as const;

/** 한 줄 시스템 표기(답 공개 턴) */
export const SYSTEM_TURN_LABELS = {
  revealed: "답을 공개했어요",
} as const;

/** 결과 아래 다음 행동 */
export const RESULT_ACTION_LABELS = {
  today: "오늘로",
} as const;

/** 확인 결과·복습 문항 아래 접힌 요약(v2.2 §6) */
export const LECTURE_SUMMARY_LABEL = "이 강의 한 장 요약";
