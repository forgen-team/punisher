import { describe, expect, it } from "vitest";
import type { CheckHistoryItem, ConceptMastery, Recommendation } from "@/src/shared/api/punisher";
import { parsePanel } from "../lib/useLecturePanel";
import { earliestReview, lastFinishedCheck, pickNextLecture } from "../lib/wrapupState";

function rec(partial: Partial<Recommendation>): Recommendation {
  return { kind: "goal_path", conceptId: "c9", conceptName: "ETag", area: "web", depth: 1, title: "t", reason: "", practicalSituation: "", estimatedMinutes: 10, ...partial };
}

function session(partial: Partial<CheckHistoryItem>): CheckHistoryItem {
  return { id: "s", kind: "check", grader: "llm", status: "finished", lectureId: "l1", itemCount: 5, doneCount: 5, unaidedCorrect: 3, createdAt: "", ...partial };
}

describe("다음 강의", () => {
  it("이어보기·이 강의·같은 개념을 건너뛴 첫 추천", () => {
    const items = [rec({ kind: "resume", lectureId: "l5" }), rec({ lectureId: "l1", conceptId: "c1" }), rec({ conceptId: "c1" }), rec({ lectureId: "l9", conceptName: "다음" })];
    expect(pickNextLecture(items, "l1", "c1")).toEqual({ title: "다음", lectureId: "l9", conceptId: "c9" });
  });
  it("없으면 null", () => {
    expect(pickNextLecture(undefined, "l1", "c1")).toBeNull();
  });
});

describe("확인 결과", () => {
  it("이 강의의 끝난 확인하기만(복습·진행 중 제외)", () => {
    const list = [session({ kind: "review" }), session({ status: "active" }), session({ lectureId: "l2" }), session({ id: "hit" })];
    expect(lastFinishedCheck(list, "l1")?.id).toBe("hit");
    expect(lastFinishedCheck([], "l1")).toBeNull();
  });
  it("다음 복습일 = 이 강의 목표 중 가장 이른 날", () => {
    const mastery = { objectives: [
      { objectiveId: "a", nextReviewAt: "2026-12-09T00:00:00Z" },
      { objectiveId: "b", nextReviewAt: "2026-12-02T00:00:00Z" },
      { objectiveId: "other", nextReviewAt: "2026-11-30T00:00:00Z" },
      { objectiveId: "c", nextReviewAt: null },
    ] } as unknown as ConceptMastery;
    expect(earliestReview(mastery, ["a", "b", "c"])?.toISOString()).toBe("2026-12-02T00:00:00.000Z");
    expect(earliestReview(undefined, ["a"])).toBeNull();
  });
});

describe("?panel", () => {
  it("toc·notes 만 받는다", () => {
    expect(parsePanel("toc")).toBe("toc");
    expect(parsePanel("notes")).toBe("notes");
    expect(parsePanel("terms")).toBeNull();
    expect(parsePanel(null)).toBeNull();
  });
});
