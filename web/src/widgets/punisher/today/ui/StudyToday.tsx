import type { ReactNode } from "react";
import { Layers } from "lucide-react";
import { Link } from "@/src/shared/lib/router";
import { Button, Menu, Section, useToast, type MenuItemDef } from "@forgen-team/ui";
import { cn } from "@/src/shared/lib/cn";
import { useActivity, useLeechAction, useLeeches, useReviewForecast } from "@/src/entities/punisher";
import { useLaunchInstant } from "@/src/features/punisher/instant-start";
import type { ActivityResponse, ForecastResponse, Leech } from "@/src/shared/api/punisher";
import { InlineError } from "@/src/shared/ui/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { activeDaysLastWeek, activityLevel, activityScore, forecastBars, forecastSummary, heatmapWeeks, type ForecastBar } from "../lib/studyStats";
import { messages } from "../i18n";

// CSS 변수 색은 /40 같은 투명도 표기가 먹지 않아(tailwind.preset.js) opacity 로 단계를 낸다.
const LEVEL_CLASS = ["bg-canvas-hover", "bg-accent-subtle", "bg-accent opacity-50", "bg-accent opacity-75", "bg-accent"] as const;

/** 히트맵 칸 크기·간격(px) — 고정해 어느 폭에서도 잘리거나 늘어나지 않게 한다. */
const CELL = 10;
const GAP = 2;
/** 예측 막대 영역 높이(px) */
const BAR_AREA = 64;
const WEEKDAY_ROWS = [1, 3, 5] as const; // 월·수·금만 이름표

/**
 * 오늘 목표 진행 + 지난 12주 히트맵 + 앞으로 14일 복습 예측(W5a). 손실 압박 없는 형태: 연속일·'끊김' 표시 없이 한 양만
 * 보이고, 쉬는 날은 "기다려 줘요". 숫자가 아직 없으면(첫 로드) 그리지 않는다.
 * 배치: 폰은 세로로 쌓고, md(768)~ 2열(진행 | 히트맵, 예측은 아래 전체 폭), xl(1280)~ 3열 [진행 | 히트맵 | 예측]이
 * 같은 높이로. 카드(variant="card")는 좌우 여백만 주므로 위아래 여백(py-4)은 여기서 둔다.
 */
export function StudyToday() {
  const t = useT(messages);
  const activity = useActivity();
  const forecast = useReviewForecast();
  const a = activity.data;
  if (!a) { return null; }
  return (
    <Section variant="card" title={t.goalTitle}>
      <div className="grid gap-5 py-4 md:grid-cols-2 xl:grid-cols-[minmax(0,1fr)_auto_minmax(0,1.3fr)] xl:gap-0">
        <TodayProgress activity={a} />
        <Heatmap activity={a} />
        <Forecast data={forecast.data} />
      </div>
      <LeechList />
    </Section>
  );
}

function ColumnTitle({ children }: { children: ReactNode }) {
  return <h3 className="text-xs font-medium text-fg-secondary">{children}</h3>;
}

function TodayProgress({ activity: a }: { activity: ActivityResponse }) {
  const t = useT(messages);
  const { today } = a;
  const reviewTotal = today.reviewsDone + today.reviewsDue;
  const done = reviewTotal === 0 || today.reviewsDue === 0;
  return (
    <div className="flex min-w-0 flex-col gap-2 xl:pr-6">
      <ColumnTitle>{t.colProgress}</ColumnTitle>
      {a.restToday ? (
        <p className="text-sm text-fg-secondary">{t.restDay}</p>
      ) : reviewTotal > 0 ? (
        <div className="flex flex-col gap-1.5">
          <p className="text-sm font-medium text-fg tabular-nums">{done ? t.allDone : t.goalReviews(today.reviewsDone, reviewTotal)}</p>
          <div className="h-1.5 w-full max-w-xs overflow-hidden rounded-full bg-canvas-hover" aria-hidden="true">
            <div className="h-full rounded-full bg-accent" style={{ width: `${Math.round((today.reviewsDone / reviewTotal) * 100)}%` }} />
          </div>
        </div>
      ) : (
        <p className="text-sm text-fg-secondary">{t.noReviewsToday}</p>
      )}
      <p className="flex flex-wrap gap-x-3 gap-y-1 text-xs text-fg-secondary">
        {(today.cardsDone > 0 || today.cardsDue > 0) && <span>{t.goalCards(today.cardsDone, today.cardsDue)}</span>}
        {today.lecturesDone > 0 && <span>{t.goalLectures(today.lecturesDone)}</span>}
      </p>
      {today.cardsDue > 0 && (
        <Button asChild variant="secondary" size="sm" className="w-fit">
          <Link href="/cards">
            <Layers className="size-4" aria-hidden="true" />
            {t.openCards(today.cardsDue)}
          </Link>
        </Button>
      )}
    </div>
  );
}

function Heatmap({ activity: a }: { activity: ActivityResponse }) {
  const t = useT(messages);
  const locale = useLocale();
  const weeks = heatmapWeeks(a.days);
  const dayLabel = (day: string) => formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "short", day: "numeric", weekday: "short" });
  const gridHeight = 7 * CELL + 6 * GAP;
  return (
    <div className="flex min-w-0 flex-col gap-2 xl:border-l xl:border-line xl:px-6">
      <ColumnTitle>{t.colHeatmap}</ColumnTitle>
      <p className="text-sm text-fg">{t.activeDays(activeDaysLastWeek(a.days))}</p>
      {/* 고정 크기 칸. 폭이 모자라면(아주 좁은 폰) 이 상자만 가로로 밀린다 — 잘리지 않는다. */}
      <div className="flex max-w-full gap-1.5 overflow-x-auto pb-0.5 [scrollbar-width:thin]">
        <div aria-hidden="true" className="relative shrink-0 text-[10px] leading-none text-fg-muted" style={{ width: 14, height: gridHeight }}>
          {WEEKDAY_ROWS.map((d) => (
            <span key={d} className="absolute left-0" style={{ top: d * (CELL + GAP) }}>{t.weekdayShort(d)}</span>
          ))}
        </div>
        <div role="img" aria-label={t.heatmapLabel} className="flex shrink-0" style={{ gap: GAP, height: gridHeight }}>
          {weeks.map((week, wi) => (
            <div key={wi} className="flex flex-col" style={{ gap: GAP }}>
              {week.map((d, di) => {
                if (!d) { return <span key={di} style={{ width: CELL, height: CELL }} />; }
                const score = activityScore(d);
                const rest = a.restDays.includes(di);
                return (
                  <span
                    key={d.day}
                    title={t.heatmapCell(dayLabel(d.day), score)}
                    style={{ width: CELL, height: CELL }}
                    className={cn("rounded-[2px]", LEVEL_CLASS[activityLevel(score)], rest && score === 0 && "opacity-50")}
                  />
                );
              })}
            </div>
          ))}
        </div>
      </div>
      <div aria-hidden="true" className="flex items-center gap-1 text-[10px] text-fg-muted">
        <span>{t.legendLess}</span>
        {LEVEL_CLASS.map((c) => <span key={c} className={cn("rounded-[2px]", c)} style={{ width: CELL, height: CELL }} />)}
        <span>{t.legendMore}</span>
        {a.restDays.length > 0 && <span className="ml-2">{t.legendRest}</span>}
      </div>
    </div>
  );
}

function Forecast({ data }: { data: ForecastResponse | undefined }) {
  const t = useT(messages);
  const locale = useLocale();
  if (!data) { return null; }
  const f = forecastSummary(data.days);
  const bars = forecastBars(data.days, BAR_AREA);
  const dayLabel = (day: string) => formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "short", day: "numeric", weekday: "short" });
  const axisLabel = (axis: ForecastBar["axis"]) => {
    if (axis === "today") { return t.axisToday; }
    if (axis === "tomorrow") { return t.axisTomorrow; }
    return axis === null ? "" : t.axisPlus(axis);
  };
  const columns = { gridTemplateColumns: `repeat(${bars.length}, minmax(0, 1fr))` };
  const empty = bars.every((b) => b.total === 0);
  return (
    <div className="flex min-w-0 flex-col gap-2 md:col-span-2 xl:col-span-1 xl:border-l xl:border-line xl:pl-6">
      <ColumnTitle>{t.forecastTitle}</ColumnTitle>
      <p className="flex flex-wrap items-baseline gap-x-2 text-sm">
        <span className="tabular-nums text-fg">{t.forecastLine(f.tomorrow, f.week, f.twoWeeks)}</span>
        {data.queued > 0 && <span className="text-xs text-fg-muted">{t.forecastQueued(data.queued)}</span>}
      </p>
      {empty ? (
        <p className="text-xs text-fg-muted">{t.forecastEmpty}</p>
      ) : (
        <div role="img" aria-label={t.forecastBars(bars.map((b) => `${dayLabel(b.day)} ${b.total}`).join(", "))} className="flex flex-col gap-1">
          {/* 막대 위 숫자 + 막대. 0 은 바닥 눈금만. 바닥선은 border-b. */}
          <div className="grid items-end gap-1 border-b border-line" style={{ ...columns, height: BAR_AREA + 14 }}>
            {bars.map((b) => (
              <div key={b.day} title={`${dayLabel(b.day)} · ${b.total}`} className="flex h-full min-w-0 flex-col items-center justify-end gap-0.5">
                {b.total > 0 && <span className="text-[10px] leading-none tabular-nums text-fg-secondary">{b.total}</span>}
                {b.total > 0 ? (
                  <span className="w-full max-w-5 rounded-t-[2px] bg-accent" style={{ height: b.height }} />
                ) : (
                  <span className="h-0.5 w-full max-w-5 bg-line-strong" />
                )}
              </div>
            ))}
          </div>
          <div aria-hidden="true" className="grid gap-1 text-[10px] text-fg-muted" style={columns}>
            {bars.map((b, i) => (
              <span key={b.day} className={cn("whitespace-nowrap", i === bars.length - 1 ? "text-right" : i === 0 ? "text-left" : "text-center")}>
                {axisLabel(b.axis)}
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

/** 자꾸 막히는 목표(반복 실패) — 다르게 설명한 강의 · 앞 개념부터 · 잠시 보류(되돌리기) · 그대로 복습. */
function LeechList() {
  const t = useT(messages);
  const leeches = useLeeches();
  const action = useLeechAction();
  const toast = useToast();
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
      onSuccess: () =>
        toast.undo(t.leechSuspended, () => {
          action.mutateAsync({ objectiveId: l.objectiveId, action: "resume" }).catch(() => toast.show(t.undoFailed, { tone: "danger" }));
        }),
    });
  }

  return (
    <div className="flex flex-col gap-2 border-t border-line py-4">
      <p className="text-sm font-medium text-fg">{t.leechTitle}</p>
      <p className="text-xs text-fg-secondary">{t.leechHint(leeches.data?.threshold ?? 4)}</p>
      <InlineError error={error ?? action.error} />
      <ul className="flex flex-col">
        {list.slice(0, 3).map((l) => {
          const menu: MenuItemDef[] = [
            { key: "prereq", label: t.leechPrereq, href: `/map?concept=${encodeURIComponent(l.conceptId)}` },
            { key: "keep", label: t.leechKeep, onSelect: () => action.mutate({ objectiveId: l.objectiveId, action: "keep" }) },
            { key: "suspend", label: t.leechSuspend, onSelect: () => suspend(l) },
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
              <Menu items={menu} label={t.leechActions(l.objectiveText.slice(0, 30))} />
            </li>
          );
        })}
      </ul>
    </div>
  );
}
