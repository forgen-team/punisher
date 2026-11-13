import { describe, expect, it } from "vitest";
import { historyDay } from "../lib/historyDate";

describe("historyDay", () => {
  it("끝 마침표를 떼어 ' · ' 구분점과 겹치지 않는다(KST 날짜)", () => {
    expect(historyDay("2026-11-12T16:30:00Z")).toBe("2026. 11. 13");
    expect(`${historyDay("2026-11-13T03:00:00Z")} · 1/3`).toBe("2026. 11. 13 · 1/3");
  });

  it("날짜가 아니면 빈 글자", () => {
    expect(historyDay("not a date")).toBe("");
  });
});
