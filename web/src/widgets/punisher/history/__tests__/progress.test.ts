import { describe, expect, it } from "vitest";
import type { RetentionBucket } from "@/src/shared/api/punisher";
import { doneOfTotal, overallRetention } from "../lib/progress";

const bucket = (attempts: number, correct: number): RetentionBucket => ({ intervalDays: 1, grader: "self", attempts, correct, rate: attempts ? correct / attempts : null });

describe("progress", () => {
  it("pools every bucket into one retention percent, or null without attempts", () => {
    expect(overallRetention([bucket(10, 8), bucket(10, 9)])).toBe(85);
    expect(overallRetention([bucket(0, 0)])).toBeNull();
    expect(overallRetention([])).toBeNull();
  });
  it("shows done over done + left", () => {
    expect(doneOfTotal(3, 12)).toBe("3/15");
    expect(doneOfTotal(0, 0)).toBe("0/0");
  });
});
