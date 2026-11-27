import { cn } from "@/src/shared/lib/cn";
import type { CheckGrader } from "@/src/shared/api/punisher";
import { useLocale, useT } from "@/src/shared/i18n";
import { messages } from "../i18n";
import { GRADER_LABELS_BY_LOCALE, MASTERY_MAX_LEVEL, masteryLabel } from "./constants";

const DOTS = Array.from({ length: MASTERY_MAX_LEVEL }, (_, index) => index + 1);

/** 목표 레벨 점(●●●○○) + 레벨 글자(L3) + 마지막 채점 주체(튜터/자기 채점)를 작게. 점만으로는 작고 다크에서 흐려 글자를 함께 둔다. */
export function MasteryDots({ level, lastGrader, className }: { level: number; lastGrader?: CheckGrader | ""; className?: string }) {
  const locale = useLocale();
  const t = useT(messages);
  const label = masteryLabel(level, locale);
  const grader = lastGrader ? GRADER_LABELS_BY_LOCALE[locale][lastGrader] : "";
  return (
    <span className={cn("inline-flex shrink-0 items-center gap-1.5", className)} title={label}>
      <span role="img" aria-label={t.masteryAria(label, grader)} className="inline-flex items-center gap-0.5">
        {DOTS.map((n) => (
          <span
            key={n}
            aria-hidden="true"
            className={cn("size-2 rounded-full", n <= level ? "bg-accent" : "border border-line-strong")}
          />
        ))}
      </span>
      <span aria-hidden="true" className="text-xs tabular-nums text-fg-secondary">
        L{level}
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
  const t = useT(messages);
  if (!count) { return null; }
  return <p className="text-xs text-fg-muted">{t.uncheckedObjectives(count)}</p>;
}
