import { useEffect, useState } from "react";
import { Button, Field, Input } from "@/src/shared/ui/kit";
import { useGoalDeadlines, useGoalPlan, useSetGoalDeadline } from "@/src/entities/punisher";
import type { DeadlinePlan } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { formatDate, useLocale, useT, type Locale } from "@/src/shared/i18n";
import { messages } from "./i18n";

function dayLabel(day: string, locale: Locale): string {
  return day ? formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "long", day: "numeric" }) : "";
}

/** 오늘(KST) YYYY-MM-DD — 날짜 입력의 최솟값. */
function todayKst(): string {
  return new Date(Date.now() + 9 * 3600_000).toISOString().slice(0, 10);
}

export function planLine(t: (typeof messages)["ko"], p: DeadlinePlan, locale: Locale): string {
  switch (p.status) {
    case "done":
      return t.planDone;
    case "overdue":
      return t.planOverdue;
    case "on_track":
      return t.planOnTrack(p.minutesPerDay, dayLabel(p.dueOn, locale), p.dailyMinutes);
    case "tight":
      return t.planTight(p.minutesPerDay, p.dailyMinutes, dayLabel(p.finishOn, locale));
    default:
      return t.planNoDeadline(p.remainingConcepts, dayLabel(p.finishOn, locale));
  }
}

/**
 * 목표 기한(W5a): 날짜를 정하면 남은 개념 × 약 20분[추정]을 쉬는 날을 뺀 날 수로 나눠 하루 필요한 시간을 보인다.
 * 하루 학습량(설정 > 학습)과 비교해 여유/빠듯만 알린다(압박 문구 없음).
 */
export function GoalDeadline({ goalId }: { goalId: string }) {
  const t = useT(messages);
  const locale = useLocale();
  const deadlines = useGoalDeadlines();
  const plan = useGoalPlan(goalId);
  const set = useSetGoalDeadline();
  const saved = deadlines.data?.[goalId] ?? "";
  const [value, setValue] = useState(saved);
  useEffect(() => setValue(saved), [saved]);

  return (
    <div className="flex flex-col gap-2 py-2">
      <div className="flex flex-wrap items-end gap-2">
        <Field label={t.deadline}>
          <Input type="date" value={value} min={todayKst()} onChange={(e) => setValue(e.target.value)}
            onBlur={() => { if (value !== saved) { set.mutate({ goalId, dueOn: value }); } }} className="w-44" />
        </Field>
        {saved && (
          <Button variant="ghost" size="sm" loading={set.isPending && set.variables?.dueOn === ""} onClick={() => set.mutate({ goalId, dueOn: "" })}>
            {t.clearDeadline}
          </Button>
        )}
      </div>
      <InlineError error={set.error ?? plan.error} />
      {plan.data && <p aria-live="polite" className="text-sm text-fg-secondary">{planLine(t, plan.data, locale)}</p>}
    </div>
  );
}
