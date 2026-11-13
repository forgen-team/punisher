import { describe, expect, it } from "vitest";
import type { CheckHistoryItem, RetentionBucket } from "@/src/shared/api/punisher";
import { historyMeta, historyTitle, metricRows, resolvedMeta } from "../lib/checkHistory";

function bucket(intervalDays: number, grader: "llm" | "self", attempts = 0, correct = 0): RetentionBucket {
  return { intervalDays, grader, attempts, correct, rate: attempts > 0 ? correct / attempts : null };
}

function session(patch: Partial<CheckHistoryItem> = {}): CheckHistoryItem {
  return {
    id: "s1", kind: "check", grader: "llm", status: "finished", conceptName: "페이지 캐시", lectureTitle: "읽기 경로",
    itemCount: 3, doneCount: 3, unaidedCorrect: 2, createdAt: "2026-11-13T03:00:00Z", finishedAt: null, ...patch,
  };
}

describe("metricRows", () => {
  it("orders intervals and shows '—' for null rates", () => {
    const rows = metricRows([bucket(30, "llm"), bucket(1, "self", 3, 2), bucket(1, "llm"), bucket(30, "self", 4, 4)]);
    expect(rows.map((r) => r.intervalDays)).toEqual([1, 30]);
    expect(rows[0].cells.self).toEqual({ rate: "67%", count: "2/3" });
    expect(rows[0].cells.llm).toEqual({ rate: "—", count: "" });
    expect(rows[1].cells.self.rate).toBe("100%");
  });

  it("fills a missing grader with '—'", () => {
    const rows = metricRows([bucket(90, "self", 2, 0)]);
    expect(rows[0].cells.self).toEqual({ rate: "0%", count: "0/2" });
    expect(rows[0].cells.llm.rate).toBe("—");
  });
});

describe("history rows", () => {
  it("titles checks with concept · lecture and reviews by kind", () => {
    expect(historyTitle(session())).toBe("확인 · 페이지 캐시 · 읽기 경로");
    expect(historyTitle(session({ lectureTitle: "페이지 캐시" }))).toBe("확인 · 페이지 캐시");
    expect(historyTitle(session({ kind: "review", conceptName: undefined, lectureTitle: undefined }))).toBe("복습");
  });

  it("shows date, n/N, unaided count and an unfinished status", () => {
    expect(historyMeta(session({ kind: "review", reviewDay: "2026-11-12", doneCount: 1, status: "abandoned" }))).toBe(
      "2026. 11. 12 · 1/3 · 혼자 맞힘 2 · 그만둠",
    );
    expect(historyMeta(session())).toMatch(/ · 3\/3 · 혼자 맞힘 2$/);
  });
});

describe("resolvedMeta", () => {
  it("shows the concept and the KST day it was fixed", () => {
    expect(resolvedMeta({ conceptName: "페이지 캐시", resolvedAt: "2026-11-12T16:30:00Z" })).toBe("페이지 캐시 · 2026. 11. 13");
    expect(resolvedMeta({ conceptName: "페이지 캐시", resolvedAt: null })).toBe("페이지 캐시");
  });
});
