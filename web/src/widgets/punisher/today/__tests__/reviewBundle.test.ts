import { describe, expect, it } from "vitest";
import type { ReviewSessionRef, TodayReview } from "@/src/shared/api/punisher";
import { recheckWithoutLecture, reviewBundleState, reviewTotal, type ReviewInputs } from "../lib/reviewBundle";

function review(patch: Partial<TodayReview> = {}): TodayReview {
  return { due: 0, overdue: 0, queued: 0, estimatedMinutes: 0, backlogged: false, lectureMinutes: 30, verify: 0, ...patch };
}

function session(patch: Partial<ReviewSessionRef> = {}): ReviewSessionRef {
  return { id: "r1", status: "active", grader: "self", total: 15, done: 6, ...patch };
}

type TodayPart = ReviewInputs["today"];
function inputs(today: TodayPart, patch: Partial<Omit<ReviewInputs, "today">> = {}): ReviewInputs {
  return { today, session: null, cardsDue: 0, restToday: false, ...patch };
}

const unchecked = (lectureId: string) => ({ lectureId, conceptId: `c-${lectureId}`, conceptName: lectureId, area: "os", title: "" });

describe("reviewBundleState — 숫자 줄", () => {
  it("합친다: 복습(+이미 앎 확인) · 확인 대기(확인하기 + 강의 있는 다시 확인) · 카드 · 종합", () => {
    const today: TodayPart = {
      review: review({ due: 10, verify: 2 }),
      unchecked: [unchecked("l1"), unchecked("l2")],
      recheck: [
        { conceptId: "a", conceptName: "A", area: "os", goalId: "g", due: false, lectureId: "l9" },
        { conceptId: "b", conceptName: "B", area: "os", goalId: "g", due: true },
        { conceptId: "c", conceptName: "C", area: "os", goalId: "g", due: false },
      ],
      synthesis: { due: true },
      session: { lectureMinutes: 27, checkMinutes: 6, reviewMinutes: 9 },
    };
    const state = reviewBundleState(inputs(today, { cardsDue: 8 }));
    expect(state).toMatchObject({ kind: "due", counts: { review: 12, confirm: 3, cards: 8, synthesis: 1, minutes: 15 }, target: { kind: "review" } });
    expect(reviewTotal(state.counts)).toBe(23);
    // 강의 없는 '다시 확인'은 추천 강의로 간다
    expect(recheckWithoutLecture(today?.recheck).map((c) => c.conceptId)).toEqual(["c"]);
  });

  it("[복습 시작] 대상: 복습이 0 이면 첫 확인 대기, 그것도 없으면 종합", () => {
    expect(reviewBundleState(inputs({ review: review(), unchecked: [unchecked("l1")], synthesis: { due: true } }))).toMatchObject({ target: { kind: "check", lectureId: "l1" } });
    expect(reviewBundleState(inputs({ review: review(), synthesis: { due: true } }))).toMatchObject({ kind: "due", target: { kind: "synthesis" } });
  });

  it("카드만 남으면 숫자 줄에 시작 대상 없음([카드로 복습]만)", () => {
    expect(reviewBundleState(inputs({ review: review() }, { cardsDue: 4 }))).toMatchObject({ kind: "due", counts: { cards: 4 }, target: { kind: "none" } });
  });
});

describe("reviewBundleState — 상태", () => {
  it("할 것 없음", () => {
    expect(reviewBundleState(inputs({ review: review() })).kind).toBe("none");
    expect(reviewBundleState(inputs(undefined)).kind).toBe("none");
  });

  it("다 함 — 오늘 세션을 마쳤으면 d/t", () => {
    expect(reviewBundleState(inputs({ review: review({ due: 3 }) }, { session: session({ status: "finished", done: 15, total: 15 }) }))).toMatchObject({ kind: "done", done: 15, total: 15 });
  });

  it("중간에 나감 — 열린 세션의 남은 수", () => {
    expect(reviewBundleState(inputs({ review: review({ due: 9 }) }, { session: session({ total: 15, done: 6 }) }))).toMatchObject({ kind: "partial", left: 9 });
  });

  it("쉬는 날 — 할 것이 있어도 쉬는 날 문구 + 시작 대상", () => {
    expect(reviewBundleState(inputs({ review: review({ due: 4 }) }, { restToday: true }))).toMatchObject({ kind: "rest", target: { kind: "review" } });
  });

  it("버린 세션은 없는 것으로 본다", () => {
    expect(reviewBundleState(inputs({ review: review({ due: 2 }) }, { session: session({ status: "abandoned" }) })).kind).toBe("due");
  });
});
