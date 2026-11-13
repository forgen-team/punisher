import { cn } from "@/src/shared/ui/kit";
import type { CheckGrader } from "@/src/shared/api/punisher";
import { GRADER_LABELS, MASTERY_MAX_LEVEL, masteryLabel } from "./constants";

const DOTS = Array.from({ length: MASTERY_MAX_LEVEL }, (_, index) => index + 1);

/** 목표 레벨 점(●●●○○) + 마지막 채점 주체(튜터/자기)를 작게. */
export function MasteryDots({ level, lastGrader, className }: { level: number; lastGrader?: CheckGrader | ""; className?: string }) {
  const label = masteryLabel(level);
  const grader = lastGrader ? GRADER_LABELS[lastGrader] : "";
  return (
    <span className={cn("inline-flex shrink-0 items-center gap-1.5", className)} title={label}>
      <span role="img" aria-label={`숙달 ${label}${grader ? `, ${grader} 채점` : ""}`} className="inline-flex items-center gap-0.5">
        {DOTS.map((n) => (
          <span
            key={n}
            aria-hidden="true"
            className={cn("size-1.5 rounded-full", n <= level ? "bg-accent" : "border border-line-strong")}
          />
        ))}
      </span>
      {grader && (
        <span aria-hidden="true" className="text-xs text-fg-muted">
          {grader}
        </span>
      )}
    </span>
  );
}

/** "새 목표 n개 미확인" — 0 이면 그리지 않는다. */
export function UncheckedObjectives({ count }: { count: number | undefined }) {
  if (!count) { return null; }
  return <p className="text-xs text-fg-muted">새 목표 {count}개 미확인</p>;
}
