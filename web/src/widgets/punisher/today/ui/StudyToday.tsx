import { Layers } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { Button, Menu, Section, buttonVariants, cn, notifyUndo, type MenuItem } from "@/src/shared/ui/kit";
import { useActivity, useLeechAction, useLeeches, useReviewForecast } from "@/src/entities/punisher";
import { useLaunchInstant } from "@/src/features/punisher/instant-start";
import type { Leech } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { activeDaysLastWeek, activityLevel, activityScore, forecastSummary, heatmapWeeks } from "../lib/studyStats";
import { messages } from "../i18n";

// CSS 변수 색은 /40 같은 투명도 표기가 먹지 않아(tailwind.preset.js) opacity 로 단계를 낸다.
const LEVEL_CLASS = ["bg-canvas-hover", "bg-accent-subtle", "bg-accent opacity-50", "bg-accent opacity-75", "bg-accent"] as const;

/**
 * 오늘 목표 진행 + 지난 12주 히트맵 + 복습 예측(W5a). 손실 압박 없는 형태: 연속일·'끊김' 표시 없이 한 양만 보이고,
 * 쉬는 날은 "기다려 줘요". 숫자가 아직 없으면(첫 로드) 그리지 않는다.
 */
export function StudyToday() {
  const t = useT(messages);
  const locale = useLocale();
  const activity = useActivity();
  const forecast = useReviewForecast();
  const a = activity.data;
  if (!a) { return null; }
  const { today } = a;
  const reviewTotal = today.reviewsDone + today.reviewsDue;
  const done = reviewTotal === 0 || today.reviewsDue === 0;
  const weeks = heatmapWeeks(a.days);
  const f = forecast.data ? forecastSummary(forecast.data.days) : null;
  const peak = Math.max(1, f?.peak ?? 1);
  const dayLabel = (day: string) => formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "short", day: "numeric", weekday: "short" });

  return (
    <Section variant="card" title={t.goalTitle}>
      <div className="flex flex-col gap-4 md:flex-row md:items-start md:justify-between">
        <div className="flex min-w-0 flex-col gap-2">
          {a.restToday ? (
            <p className="text-sm text-fg-secondary">{t.restDay}</p>
          ) : (
            <>
              {reviewTotal > 0 && (
                <div className="flex flex-col gap-1">
                  <p className="text-sm font-medium text-fg">{done ? t.allDone : t.goalReviews(today.reviewsDone, reviewTotal)}</p>
                  <div className="h-1.5 w-48 max-w-full overflow-hidden rounded-full bg-canvas-hover" aria-hidden="true">
                    <div className="h-full rounded-full bg-accent" style={{ width: `${Math.round((today.reviewsDone / reviewTotal) * 100)}%` }} />
                  </div>
                </div>
              )}
            </>
          )}
          <p className="flex flex-wrap gap-x-3 text-xs text-fg-secondary">
            {(today.cardsDone > 0 || today.cardsDue > 0) && <span>{t.goalCards(today.cardsDone, today.cardsDue)}</span>}
            {today.lecturesDone > 0 && <span>{t.goalLectures(today.lecturesDone)}</span>}
            <span>{t.activeDays(activeDaysLastWeek(a.days))}</span>
          </p>
          {today.cardsDue > 0 && (
            <Link href="/cards" className={buttonVariants({ variant: "secondary", size: "sm", className: "w-fit" })}>
              <Layers className="size-4" aria-hidden="true" />
              {t.openCards(today.cardsDue)}
            </Link>
          )}
        </div>
        <div className="flex flex-col gap-1">
          <div role="img" aria-label={t.heatmapLabel} className="flex gap-0.5">
            {weeks.map((week, wi) => (
              <div key={wi} className="flex flex-col gap-0.5">
                {week.map((d, di) => {
                  if (!d) { return <span key={di} className="size-2.5" />; }
                  const score = activityScore(d);
                  const rest = a.restDays.includes(di);
                  return (
                    <span
                      key={d.day}
                      title={t.heatmapCell(dayLabel(d.day), score)}
                      className={cn("size-2.5 rounded-[2px]", LEVEL_CLASS[activityLevel(score)], rest && score === 0 && "opacity-50")}
                    />
                  );
                })}
              </div>
            ))}
          </div>
          <p className="text-[11px] text-fg-muted">{t.heatmapLabel}</p>
        </div>
      </div>
      {f && forecast.data && (
        <div className="mt-4 flex flex-col gap-1 border-t border-line pt-3">
          <p className="flex flex-wrap items-baseline gap-x-2 text-sm">
            <span className="font-medium text-fg">{t.forecastTitle}</span>
            <span className="tabular-nums text-fg-secondary">{t.forecastLine(f.tomorrow, f.week, f.twoWeeks)}</span>
            {forecast.data.queued > 0 && <span className="text-xs text-fg-muted">{t.forecastQueued(forecast.data.queued)}</span>}
          </p>
          <div
            role="img"
            aria-label={t.forecastBars(forecast.data.days.map((d) => d.objectives + d.cards).join(", "))}
            className="flex h-8 items-end gap-0.5"
          >
            {forecast.data.days.map((d) => (
              <span
                key={d.day}
                title={`${dayLabel(d.day)} · ${d.objectives + d.cards}`}
                className="w-2.5 rounded-t-[2px] bg-accent opacity-60"
                style={{ height: `${Math.max(2, Math.round(((d.objectives + d.cards) / peak) * 32))}px` }}
              />
            ))}
          </div>
        </div>
      )}
      <LeechList />
    </Section>
  );
}

/** 자꾸 막히는 목표(반복 실패) — 다르게 설명한 강의 · 앞 개념부터 · 잠시 보류(되돌리기) · 그대로 복습. */
function LeechList() {
  const t = useT(messages);
  const leeches = useLeeches();
  const action = useLeechAction();
  const { launch, isPending, error } = useLaunchInstant();
  const list = leeches.data?.leeches ?? [];
  if (list.length === 0) { return null; }

  async function relearn(l: Leech) {
    try {
      await launch({ conceptId: l.conceptId, query: t.leechQuery(l.objectiveText) });
    } catch (err) {
      // 실패 문구는 위 InlineError(error) 가 보인다.
      console.warn("punisher: relearn launch failed", err);
    }
  }

  function suspend(l: Leech) {
    action.mutate({ objectiveId: l.objectiveId, action: "suspend" }, {
      onSuccess: () => notifyUndo(t.leechSuspended, () => action.mutateAsync({ objectiveId: l.objectiveId, action: "resume" })),
    });
  }

  return (
    <div className="mt-4 flex flex-col gap-2 border-t border-line pt-3">
      <p className="text-sm font-medium text-fg">{t.leechTitle}</p>
      <p className="text-xs text-fg-secondary">{t.leechHint(leeches.data?.threshold ?? 4)}</p>
      <InlineError error={error ?? action.error} />
      <ul className="flex flex-col">
        {list.slice(0, 3).map((l) => {
          const menu: MenuItem[] = [
            { key: "prereq", label: t.leechPrereq, href: `/map?concept=${encodeURIComponent(l.conceptId)}` },
            { key: "keep", label: t.leechKeep, onSelect: () => action.mutate({ objectiveId: l.objectiveId, action: "keep" }) },
            { key: "suspend", label: t.leechSuspend, onSelect: () => suspend(l), separatorBefore: true },
          ];
          return (
            <li key={l.objectiveId} className="flex items-center gap-2 border-b border-line py-2 last:border-b-0">
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm text-fg">{l.objectiveText}</p>
                <p className="truncate text-xs text-fg-muted">{l.conceptName}</p>
              </div>
              <Button variant="ghost" size="sm" loading={isPending} onClick={() => void relearn(l)}>
                {t.leechRelearn}
              </Button>
              <Menu items={menu} label={t.leechActions(l.objectiveText.slice(0, 30))} linkAs={Link} />
            </li>
          );
        })}
      </ul>
    </div>
  );
}
