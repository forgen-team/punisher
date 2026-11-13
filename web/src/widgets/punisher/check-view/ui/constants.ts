import type { BadgeTone } from "@/src/shared/ui/kit";
import type { CheckOutcome, CheckSessionStatus, CheckVerdict } from "@/src/shared/api/punisher";

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
};

/** 이의 제기 재채점 결과 */
export const DISPUTE_RESULT_LABELS: Record<"accepted" | "rejected", string> = {
  accepted: "이의 받아들임",
  rejected: "원래 판정 유지",
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
