import { describe, expect, it } from "vitest";
import { activeDaysLastWeek, activityLevel, activityScore, forecastSummary, heatmapWeeks, weekday } from "../lib/studyStats";

const day = (d: string, reviews = 0, lectures = 0, cards = 0) => ({ day: d, reviews, lectures, cards });

describe("study stats", () => {
  it("scores and levels activity", () => {
    expect(activityScore(day("2026-11-25", 2, 1, 1))).toBe(6);
    expect([0, 1, 5, 12, 30].map(activityLevel)).toEqual([0, 1, 2, 3, 4]);
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
