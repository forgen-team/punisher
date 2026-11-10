import { describe, expect, it } from "vitest";
import { placeOverlay } from "../overlayPlacement";

const base = { width: 380, viewportW: 1440, viewportH: 800, topInset: 0, bottomInset: 0, gap: 8, margin: 12 };

describe("placeOverlay", () => {
  it("sits below the target when everything fits", () => {
    expect(placeOverlay({ ...base, anchor: { top: 100, bottom: 120, left: 300 }, naturalHeight: 200 }).top).toBe(128);
  });

  it("goes above when it would be cut at the bottom (설명이 늦게 도착해 커져도)", () => {
    const p = placeOverlay({ ...base, anchor: { top: 730, bottom: 750, left: 300 }, naturalHeight: 375 });
    expect(p.top).toBe(347);
    expect(p.top + p.maxHeight).toBeLessThanOrEqual(788);
  });

  it("stays clear of the bottom tab bar", () => {
    const p = placeOverlay({ ...base, bottomInset: 56, anchor: { top: 600, bottom: 620, left: 10 }, naturalHeight: 200 });
    expect(p.top + 200).toBeLessThanOrEqual(800 - 56 - 12);
  });

  it("shows as much as the screen allows when neither side fits", () => {
    const p = placeOverlay({ ...base, anchor: { top: 400, bottom: 420, left: 300 }, naturalHeight: 600 });
    expect(p.top).toBeGreaterThanOrEqual(12);
    expect(p.top + 600).toBeLessThanOrEqual(788);
    const huge = placeOverlay({ ...base, anchor: { top: 400, bottom: 420, left: 300 }, naturalHeight: 2000 });
    expect(huge.top).toBe(12);
    expect(huge.maxHeight).toBe(776);
  });

  it("keeps the panel inside the screen horizontally", () => {
    expect(placeOverlay({ ...base, anchor: { top: 100, bottom: 120, left: 1300 }, naturalHeight: 100 }).left).toBe(1048);
  });
});
