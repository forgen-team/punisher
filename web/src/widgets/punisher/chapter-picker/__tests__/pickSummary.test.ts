import { describe, expect, it } from "vitest";
import { pickSummary } from "../lib/pickSummary";

describe("pickSummary", () => {
  it("고른 카드 수와 직접 입력 여부를 한 줄로", () => {
    expect(pickSummary(2, "")).toBe("선택 2개");
    expect(pickSummary(2, "쿠버네티스")).toBe("선택 2개 · 직접 입력");
    expect(pickSummary(0, "쿠버네티스")).toBe("직접 입력");
  });

  it("아무것도 고르지 않으면 빈 글자(공백 입력 포함)", () => {
    expect(pickSummary(0, "   ")).toBe("");
  });
});
