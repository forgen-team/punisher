import type { BadgeTone } from "@/src/shared/ui/kit";
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
};
