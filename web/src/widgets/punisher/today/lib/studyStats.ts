import type { ActivityDay, ForecastDay } from "@/src/shared/api/punisher";

/** 한 날의 활동 양(히트맵 칸 진하기): 강의 완료는 복습 3개 몫. */
export function activityScore(d: ActivityDay): number {
  return d.reviews + d.cards + d.lectures * 3;
}

/** 0~4 단계(0 = 쉼). 손실 압박 없이 '얼마나 했는지'만 보인다. */
export function activityLevel(score: number): 0 | 1 | 2 | 3 | 4 {
  if (score <= 0) { return 0; }
  if (score <= 3) { return 1; }
  if (score <= 8) { return 2; }
  if (score <= 15) { return 3; }
  return 4;
}

/** 날짜(YYYY-MM-DD)의 요일 0=일 … 6=토(날짜만 보고 정한다 — 시간대 영향 없음). */
export function weekday(day: string): number {
  const [y, m, d] = day.split("-").map(Number);
  return new Date(Date.UTC(y ?? 1970, (m ?? 1) - 1, d ?? 1)).getUTCDay();
}

/**
 * 히트맵 열(주): 일요일 시작 7칸씩, 첫 주 앞은 빈칸(null). days 는 오래된 날부터.
 */
export function heatmapWeeks(days: ActivityDay[]): Array<Array<ActivityDay | null>> {
  if (days.length === 0) { return []; }
  const cells: Array<ActivityDay | null> = [...Array(weekday(days[0]?.day ?? "")).fill(null), ...days];
  const weeks: Array<Array<ActivityDay | null>> = [];
  for (let i = 0; i < cells.length; i += 7) { weeks.push(cells.slice(i, i + 7)); }
  return weeks;
}

/** 최근 7일 중 무엇이든 한 날 수. */
export function activeDaysLastWeek(days: ActivityDay[]): number {
  return days.slice(-7).filter((d) => activityScore(d) > 0).length;
}

/** 예측 요약: 내일 · 7일 합 · 14일 합(오늘 포함 첫 칸은 지금 밀린 것까지). */
export function forecastSummary(days: ForecastDay[]): { today: number; tomorrow: number; week: number; twoWeeks: number; peak: number } {
  const total = (d: ForecastDay) => d.objectives + d.cards;
  const sum = (list: ForecastDay[]) => list.reduce((n, d) => n + total(d), 0);
  return {
    today: days[0] ? total(days[0]) : 0,
    tomorrow: days[1] ? total(days[1]) : 0,
    week: sum(days.slice(0, 7)),
    twoWeeks: sum(days.slice(0, 14)),
    peak: days.reduce((m, d) => Math.max(m, total(d)), 0),
  };
}
