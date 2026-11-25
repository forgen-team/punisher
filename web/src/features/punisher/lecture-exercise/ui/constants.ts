import type { ExerciseVerdict } from "@/src/shared/api/punisher";

export const VERDICT_TONE: Record<Exclude<ExerciseVerdict, "">, string> = {
  correct: "text-success",
  partial: "text-warning",
  incorrect: "text-danger",
  skipped: "text-fg-muted",
  matched: "text-success",
  different: "text-warning",
  unreadable: "text-fg-muted",
  remembered: "text-success",
  forgot: "text-warning",
  written: "text-fg-muted",
};
