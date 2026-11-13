import { describe, expect, it } from "vitest";
import type { ReviewSessionRef, TodayReview } from "@/src/shared/api/punisher";
import { reviewCardState } from "../lib/reviewCard";

function review(patch: Partial<TodayReview> = {}): TodayReview {
  return { due: 0, overdue: 0, queued: 0, estimatedMinutes: 0, backlogged: false, lectureMinutes: 30, ...patch };
}

function session(patch: Partial<ReviewSessionRef> = {}): ReviewSessionRef {
  return { id: "r1", status: "active", grader: "self", total: 3, done: 1, ...patch };
}

describe("reviewCardState", () => {
  it("hides without review data, or with nothing due and no finished session", () => {
    expect(reviewCardState(undefined, null)).toEqual({ kind: "hidden" });
    expect(reviewCardState(review(), null)).toEqual({ kind: "hidden" });
    expect(reviewCardState(review(), session({ status: "abandoned" }))).toEqual({ kind: "hidden" });
  });

  it("shows due count, minutes and the time split with '복습 시작'", () => {
    expect(reviewCardState(review({ due: 4, estimatedMinutes: 4 }), null)).toEqual({
      kind: "due", due: 4, minutes: 4, lectureMinutes: 30, backlog: null, action: "start",
    });
    expect(reviewCardState(review({ due: 2, estimatedMinutes: 2 }), session())).toMatchObject({ action: "start" });
  });

  it("offers '오늘 더 하기' when today's session is finished and reviews remain", () => {
    expect(reviewCardState(review({ due: 5, estimatedMinutes: 5 }), session({ status: "finished", done: 3 }))).toMatchObject({
      kind: "due", action: "more",
    });
  });

  it("shows the backlog (overdue, else due) only when backlogged", () => {
    expect(reviewCardState(review({ due: 40, overdue: 25, backlogged: true }), null)).toMatchObject({ backlog: 25 });
    expect(reviewCardState(review({ due: 40, overdue: 0, backlogged: true }), null)).toMatchObject({ backlog: 40 });
    expect(reviewCardState(review({ due: 8, overdue: 6 }), null)).toMatchObject({ backlog: null });
  });

  it("collapses to done when nothing is due and today's session is finished", () => {
    expect(reviewCardState(review(), session({ status: "finished", total: 3, done: 3 }))).toEqual({ kind: "done", done: 3, total: 3 });
  });
});
