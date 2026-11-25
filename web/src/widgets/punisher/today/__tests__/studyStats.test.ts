import { describe, expect, it } from "vitest";
import { FORECAST_MIN_BAR, activeDaysLastWeek, activityLevel, activityScore, forecastBars, forecastSummary, heatmapWeeks, weekday } from "../lib/studyStats";

const day = (d: string, reviews = 0, lectures = 0, cards = 0) => ({ day: d, reviews, lectures, cards });

describe("study stats", () => {
  it("scores and levels activity", () => {
    expect(activityScore(day("2026-11-25", 2, 1, 1))).toBe(6);
    expect([0, 1, 5, 12, 30].map((n) => activityLevel(n))).toEqual([0, 1, 2, 3, 4]);
  });
  it("lays weeks from Sunday", () => {
    expect(weekday("2026-11-25")).toBe(3);
    const weeks = heatmapWeeks([day("2026-11-25"), day("2026-11-26"), day("2026-11-27"), day("2026-11-28"), day("2026-11-29")]);
    expect(weeks).toHaveLength(2);
    expect(weeks[0]?.slice(0, 3)).toEqual([null, null, null]);
    expect(weeks[1]?.[0]?.day).toBe("2026-11-29");
  });
  it("counts active days of the last week", () => {
    expect(activeDaysLastWeek([day("a", 1), day("b"), day("c", 0, 1)])).toBe(2);
  });
  it("summarises the forecast", () => {
    const days = Array.from({ length: 14 }, (_, i) => ({ day: `d${i}`, objectives: i === 0 ? 3 : 1, cards: i === 1 ? 2 : 0 }));
    expect(forecastSummary(days)).toEqual({ today: 3, tomorrow: 3, week: 11, twoWeeks: 18, peak: 3 });
  });
});

describe("forecast bars", () => {
  const days = (totals: number[]) => totals.map((n, i) => ({ day: `d${i}`, objectives: n, cards: 0 }));
  it("keeps small values readable: a single 1 is not a full bar, zeros are flat", () => {
    const bars = forecastBars(days([0, 1, 0]), 64);
    expect(bars.map((b) => b.height)).toEqual([0, 16, 0]);
  });
  it("gives a minimum height to non-zero bars and scales to the peak", () => {
    const bars = forecastBars(days([1, 40, 0]), 64);
    expect(bars[0]?.height).toBe(FORECAST_MIN_BAR);
    expect(bars[1]?.height).toBe(64);
  });
  it("labels today, tomorrow, +7 and +14", () => {
    const bars = forecastBars(days(Array.from({ length: 15 }, () => 0)), 64);
    expect(bars.flatMap((b, i) => (b.axis === null ? [] : [[i, b.axis]]))).toEqual([[0, "today"], [1, "tomorrow"], [7, 7], [14, 14]]);
  });
});
