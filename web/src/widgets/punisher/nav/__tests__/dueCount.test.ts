import { describe, expect, it } from "vitest";
import { dueTotal } from "../lib/useDueCount";

describe("section badge total", () => {
  it("adds reviews, pending checks and cards", () => {
    expect(dueTotal({ restToday: false, reviewsDue: 12, cardsDue: 8, unchecked: 3 })).toBe(23);
  });
  it("is 0 on a rest day", () => {
    expect(dueTotal({ restToday: true, reviewsDue: 12, cardsDue: 8, unchecked: 3 })).toBe(0);
  });
  it("ignores negative numbers", () => {
    expect(dueTotal({ restToday: false, reviewsDue: -1, cardsDue: 0, unchecked: 2 })).toBe(2);
  });
});
