import { describe, expect, it } from "vitest";
import type { ChartData } from "../lib/blockData";
import { barRect, formatTick, layoutChart, linePath, linearScale, logScale } from "../lib/chartScale";

describe("linearScale", () => {
  it("includes 0 and rounds the domain to nice steps", () => {
    const s = linearScale([3, 87], [100, 0]);
    expect(s.ticks.map((t) => t.value)).toEqual([0, 20, 40, 60, 80, 100]);
    expect(s.toPx(0)).toBe(100);
    expect(s.toPx(100)).toBe(0);
    expect(s.toPx(50)).toBe(50);
  });
  it("handles negatives and a flat series", () => {
    expect(linearScale([-5, 5], [0, 1]).ticks[0].value).toBe(-6);
    expect(linearScale([0, 0], [0, 1]).ticks.length).toBeGreaterThan(1);
  });
});

describe("logScale", () => {
  it("puts powers of ten at even spacing", () => {
    const s = logScale([1, 10_000], [400, 0]);
    expect(s.ticks.map((t) => t.value)).toEqual([1, 10, 100, 1000, 10_000]);
    expect(s.toPx(100)).toBeCloseTo(200);
    expect(s.baseline).toBe(1);
  });
  it("thins ticks over many decades but keeps the top", () => {
    const s = logScale([1, 1e12], [0, 1]);
    expect(s.ticks.length).toBeLessThanOrEqual(7);
    expect(s.ticks[s.ticks.length - 1].value).toBe(1e12);
  });
  it("widens a single-decade domain", () => {
    expect(logScale([2, 3], [0, 1]).ticks.map((t) => t.value)).toEqual([1, 10]);
  });
});

describe("formatTick", () => {
  it("compacts large numbers", () => {
    expect(formatTick(1000)).toBe("1,000");
    expect(formatTick(1_000_000)).toBe("100만");
  });
});

describe("layoutChart", () => {
  const data: ChartData = {
    kind: "bar",
    title: "",
    x: { label: "n", values: ["a", "b"] },
    y: { label: "", scale: "linear" },
    series: [
      { name: "s1", values: [10, 20] },
      { name: "s2", values: [5, 15] },
    ],
    note: "",
  };
  it("spaces categories evenly inside the plot", () => {
    const layout = layoutChart(data, 400, 200);
    expect(layout.xPos[1] - layout.xPos[0]).toBeCloseTo(layout.band);
    expect(layout.xPos[0]).toBeGreaterThan(layout.plot.left);
  });
  it("draws grouped bars from the baseline", () => {
    const layout = layoutChart(data, 400, 200);
    const a = barRect(0, 2, 0, 10, layout);
    const b = barRect(1, 2, 0, 5, layout);
    expect(b.x).toBeGreaterThan(a.x);
    expect(a.y + a.height).toBeCloseTo(layout.y.toPx(0));
    expect(a.height).toBeGreaterThan(b.height);
  });
  it("builds a line path through every point", () => {
    const layout = layoutChart({ ...data, kind: "line" }, 400, 200);
    expect(linePath([10, 20], layout).match(/[ML]/g)).toEqual(["M", "L"]);
  });
});
