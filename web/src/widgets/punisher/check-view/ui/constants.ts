/** 앱 배지 색 이름 — ui Badge tone 으로는 BADGE_TONE 으로 바꿔 넘긴다. */
export type BadgeTone = "neutral" | "accent" | "success" | "warning" | "danger";

export const BADGE_TONE = { neutral: "neutral", accent: "brand", success: "positive", warning: "warning", danger: "critical" } as const;
import type { CheckOutcome, CheckVerdict } from "@/src/shared/api/punisher";

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
  skipped: "neutral",
};
