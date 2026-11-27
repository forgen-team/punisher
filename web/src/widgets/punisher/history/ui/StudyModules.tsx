import { Button, ErrorState, Section, Skeleton } from "@forgen-team/ui";
import { Link } from "@/src/shared/lib/router";
import { cn } from "@/src/shared/lib/cn";
import { useActivity, useMasteryMetrics, useReviewForecast, useStudyPrefs } from "@/src/entities/punisher";
import { describePunisherError, type ActivityResponse, type ForecastResponse } from "@/src/shared/api/punisher";
import { formatDate, useLocale, useT } from "@/src/shared/i18n";
import { activeDaysLastWeek, activityLevel, activityScore, forecastBars, forecastSummary, heatmapWeeks, type ForecastBar } from "../lib/studyStats";
import { doneOfTotal, overallRetention } from "../lib/progress";
import { messages } from "../i18n";

// CSS 변수 색은 /40 같은 투명도 표기가 먹지 않아(tailwind.preset.js) opacity 로 단계를 낸다.
const LEVEL_CLASS = ["bg-canvas-hover", "bg-accent-subtle", "bg-accent opacity-50", "bg-accent opacity-75", "bg-accent"] as const;

/** 히트맵 칸 크기·간격(px) — 고정해 어느 폭에서도 잘리거나 늘어나지 않게 한다. */
const CELL = 10;
const GAP = 3;
/** 예측 막대 영역 높이(px) */
const BAR_AREA = 64;
const WEEKDAY_ROWS = [1, 3, 5] as const; // 월·수·금만 이름표

function Kpi({ value, label }: { value: string; label: string }) {
  return (
    <div className="min-w-0">
      <div className="text-xl font-semibold tabular-nums text-fg">{value}</div>
      <div className="text-xs text-fg-muted">{label}</div>
    </div>
  );
}

/** 모듈 한 칸의 불러오는 중·실패 — 제목은 늘 보이고, 실패를 숨기지 않는다. */
function ModuleState({ title, error, onRetry }: { title: string; error?: unknown; onRetry: () => void }) {
  return (
    <Section variant="card" title={title}>
      {error ? <ErrorState title={describePunisherError(error)} onRetry={onRetry} /> : <Skeleton lines={2} />}
    </Section>
  );
}

/** 오늘 진행 한 줄 — 복습 n/N · 카드 n/N · 강의 완료 n편 · 기억 유지 n%(목표 n%). */
export function ProgressModule() {
  const t = useT(messages);
  const activity = useActivity();
  const metrics = useMasteryMetrics();
  const prefs = useStudyPrefs();
  const a = activity.data;
  if (!a) { return <ModuleState title={t.progressTitle} error={activity.error} onRetry={() => void activity.refetch()} />; }
  const retention = metrics.data ? overallRetention(metrics.data.buckets) : null;
  const target = prefs.data ? t.kpiRetentionTarget(Math.round(prefs.data.targetRetention * 100)) : t.kpiRetention;
  return (
    <Section variant="card" title={t.progressTitle} description={a.restToday ? t.restDay : undefined}>
      <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
        <Kpi value={doneOfTotal(a.today.reviewsDone, a.today.reviewsDue)} label={t.kpiReview} />
        <Kpi value={doneOfTotal(a.today.cardsDone, a.today.cardsDue)} label={t.kpiCards} />
        <Kpi value={t.kpiLecturesValue(a.today.lecturesDone)} label={t.kpiLectures} />
        <Kpi value={retention === null ? "—" : `${retention}%`} label={target} />
      </div>
    </Section>
  );
}

/** 지난 12주 히트맵 + 앞으로 14일 복습 막대 — 데스크톱은 2열. */
export function ActivityModules() {
  return (
    <div className="grid gap-4 md:grid-cols-2 md:items-start">
      <HeatmapModule />
      <ForecastModule />
    </div>
  );
}

function HeatmapModule() {
  const t = useT(messages);
  const activity = useActivity();
  if (!activity.data) { return <ModuleState title={t.heatmapTitle} error={activity.error} onRetry={() => void activity.refetch()} />; }
  return (
    <Section variant="card" title={t.heatmapTitle} description={t.activeDays(activeDaysLastWeek(activity.data.days))}>
      <Heatmap activity={activity.data} />
    </Section>
  );
}

function Heatmap({ activity: a }: { activity: ActivityResponse }) {
  const t = useT(messages);
  const locale = useLocale();
  const weeks = heatmapWeeks(a.days);
  const dayLabel = (day: string) => formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "short", day: "numeric", weekday: "short" });
  const gridHeight = 7 * CELL + 6 * GAP;
  return (
    <div className="flex min-w-0 flex-col gap-2">
      {/* 고정 크기 칸. 폭이 모자라면(아주 좁은 폰) 이 상자만 가로로 밀린다 — 잘리지 않는다. */}
      <div className="flex max-w-full gap-2 overflow-x-auto pb-1 [scrollbar-width:thin]">
        <div aria-hidden="true" className="relative shrink-0 text-xs leading-none text-fg-muted" style={{ width: 14, height: gridHeight }}>
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
                    className={cn(LEVEL_CLASS[activityLevel(score)], rest && score === 0 && "opacity-50")}
                  />
                );
              })}
            </div>
          ))}
        </div>
      </div>
      <div aria-hidden="true" className="flex items-center gap-1 text-xs text-fg-muted">
        <span>{t.legendLess}</span>
        {LEVEL_CLASS.map((c) => <span key={c} className={c} style={{ width: CELL, height: CELL }} />)}
        <span>{t.legendMore}</span>
        {a.restDays.length > 0 && <span className="ml-2">{t.legendRest}</span>}
      </div>
    </div>
  );
}

function ForecastModule() {
  const t = useT(messages);
  const forecast = useReviewForecast();
  const data = forecast.data;
  if (!data) { return <ModuleState title={t.forecastTitle} error={forecast.error} onRetry={() => void forecast.refetch()} />; }
  const f = forecastSummary(data.days);
  return (
    <Section
      variant="card"
      title={t.forecastTitle}
      description={[t.forecastLine(f.tomorrow, f.week, f.twoWeeks), data.queued > 0 ? t.forecastQueued(data.queued) : ""].filter(Boolean).join(" · ")}
    >
      <Forecast data={data} />
    </Section>
  );
}

function Forecast({ data }: { data: ForecastResponse }) {
  const t = useT(messages);
  const locale = useLocale();
  const bars = forecastBars(data.days, BAR_AREA);
  const dayLabel = (day: string) => formatDate(new Date(`${day}T00:00:00+09:00`), locale, { month: "short", day: "numeric", weekday: "short" });
  const axisLabel = (axis: ForecastBar["axis"]) => {
    if (axis === "today") { return t.axisToday; }
    // 내일 칸은 이름표 없이 둔다(오늘 이름표와 겹친다).
    return axis === null || axis === "tomorrow" ? "" : t.axisPlus(axis);
  };
  const columns = { gridTemplateColumns: `repeat(${bars.length}, minmax(0, 1fr))` };
  if (bars.every((b) => b.total === 0)) { return <p className="text-xs text-fg-muted">{t.forecastEmpty}</p>; }
  return (
    <div role="img" aria-label={t.forecastBars(bars.map((b) => `${dayLabel(b.day)} ${b.total}`).join(", "))} className="flex flex-col gap-1">
      {/* 막대 위 숫자 + 막대. 0 은 바닥 눈금만. 바닥선은 border-b. */}
      <div className="grid items-end gap-1 border-b border-line" style={{ ...columns, height: BAR_AREA + 14 }}>
        {bars.map((b) => (
          <div key={b.day} title={`${dayLabel(b.day)} · ${b.total}`} className="flex h-full min-w-0 flex-col items-center justify-end gap-1">
            {b.total > 0 && <span className="text-xs leading-none tabular-nums text-fg-secondary">{b.total}</span>}
            {b.total > 0 ? <span className="w-full bg-accent" style={{ height: b.height }} /> : <span className="h-0.5 w-full bg-line-strong" />}
          </div>
        ))}
      </div>
      <div aria-hidden="true" className="grid gap-1 text-xs text-fg-muted" style={columns}>
        {bars.map((b, i) => (
          <span key={b.day} className={cn("whitespace-nowrap", i === bars.length - 1 ? "text-right" : i === 0 ? "text-left" : "text-center")}>
            {axisLabel(b.axis)}
          </span>
        ))}
      </div>
    </div>
  );
}

/** 카드 관리 링크(모듈 머리 동작) */
export function ManageCardsLink({ count }: { count: number | undefined }) {
  const t = useT(messages);
  return (
    <Button asChild variant="ghost" size="sm">
      <Link href="/cards?tab=all">{count === undefined ? t.manageCardsPlain : t.manageCards(count)}</Link>
    </Button>
  );
}
