import { describe, expect, it } from "vitest";
import {
  APP_HEADER_STATE_ATTR,
  HEADER_SHOW_ZONE,
  HEADER_TOGGLE_DELTA,
  nextHeaderState,
  setAppHeaderHidden,
  type HeaderScrollState,
} from "../appHeaderCollapse";

function run(ys: number[], start: HeaderScrollState = { hidden: false, anchorY: 0 }): boolean[] {
  let state = start;
  return ys.map((y) => {
    state = nextHeaderState(state, y);
    return state.hidden;
  });
}

describe("셸 상단 바 숨김(스크롤 방향)", () => {
  it("stays visible near the top however the reader scrolls", () => {
    expect(run([20, 60, HEADER_SHOW_ZONE, 40])).toEqual([false, false, false, false]);
  });

  it("hides after scrolling down past the threshold, even in small steps", () => {
    const y0 = HEADER_SHOW_ZONE + 100;
    const steps = [y0, y0 + 4, y0 + 8, y0 + HEADER_TOGGLE_DELTA + 1];
    expect(run(steps, { hidden: false, anchorY: y0 })).toEqual([false, false, false, true]);
  });

  it("shows again only after scrolling up more than the threshold", () => {
    const y0 = 1000;
    expect(run([y0 + 50, y0 + 45, y0 + 40, y0 + 50 - HEADER_TOGGLE_DELTA - 1], { hidden: true, anchorY: y0 })).toEqual([true, true, true, false]);
  });

  it("ignores small bounces at the end of a fling", () => {
    expect(run([500, 520, 515, 525], { hidden: false, anchorY: 300 })).toEqual([true, true, true, true]);
    expect(run([500, 495, 505, 498], { hidden: false, anchorY: 500 })).toEqual([false, false, false, false]);
  });

  it("shows the bar when scrolled back into the top zone even if it was hidden", () => {
    expect(nextHeaderState({ hidden: true, anchorY: 2000 }, HEADER_SHOW_ZONE - 1).hidden).toBe(false);
  });
});

describe("셸 상단 바 숨김 표시(<html> 속성)", () => {
  it("marks <html> with data-app-header-state and leaves the AppFrame bar to the CSS", () => {
    setAppHeaderHidden(true);
    expect(APP_HEADER_STATE_ATTR).toBe("data-app-header-state");
    expect(document.documentElement.getAttribute("data-app-header-state")).toBe("hidden");
    setAppHeaderHidden(false);
    expect(document.documentElement.hasAttribute("data-app-header-state")).toBe(false);
  });
});
