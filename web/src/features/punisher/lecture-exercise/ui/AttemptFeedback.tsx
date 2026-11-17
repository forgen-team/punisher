import { cn } from "@/src/shared/ui/kit";
import { EXERCISE_POLL_LIMIT_MS } from "@/src/entities/punisher";
import type { ExerciseAttempt } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { VERDICT_LABELS, messages } from "../i18n";
import { VERDICT_TONE } from "./constants";

function timedOut(a: ExerciseAttempt): boolean {
  return a.status === "pending" && Date.now() - new Date(a.createdAt).getTime() >= EXERCISE_POLL_LIMIT_MS;
}

/** 튜터 평 — 쓰는 동안 한 줄 자리표시, 실패·시간 초과면 짧은 안내. */
export function AttemptFeedback({ attempt }: { attempt: ExerciseAttempt }) {
  const t = useT(messages);
  const locale = useLocale();
  if (attempt.verdict === "skipped") { return null; }
  if (attempt.status === "failed" || timedOut(attempt)) {
    return <p role="status" className="mt-2 text-sm text-fg-muted">{t.gradeFailed}</p>;
  }
  if (attempt.status === "pending") {
    return (
      <p role="status" aria-live="polite" className="mt-2 flex items-center gap-2 text-sm text-fg-muted">
        <span className="h-3 w-24 animate-pulse rounded bg-canvas-muted" aria-hidden="true" />
        {t.grading}
      </p>
    );
  }
  return (
    <div role="status" aria-live="polite" className="mt-2 text-sm">
      {attempt.verdict && (
        <p className={cn("font-semibold", VERDICT_TONE[attempt.verdict])}>{VERDICT_LABELS[locale][attempt.verdict]}</p>
      )}
      <p className="mt-0.5 break-keep text-fg-secondary">{attempt.feedback}</p>
    </div>
  );
}
